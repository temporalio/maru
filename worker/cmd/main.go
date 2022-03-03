// The MIT License
//
// Copyright (c) 2021 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"

	"github.com/temporalio/maru/bench"
	"github.com/temporalio/maru/common"
	"github.com/temporalio/maru/target/basic"
	"github.com/uber-go/tally/v4/prometheus"
	sdktally "go.temporal.io/sdk/contrib/tally"
)

func main() {
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	logger := NewZapAdapter(zapLogger)

	logger.Info("Zap logger created")
	namespace := common.GetEnvOrDefaultString(logger, "NAMESPACE", client.DefaultNamespace)
	hostPort := common.GetEnvOrDefaultString(logger, "FRONTEND_ADDRESS", client.DefaultHostPort)
	skipNamespaceCreation := common.GetEnvOrDefaultBool(logger, "SKIP_NAMESPACE_CREATION", false)

	tlsConfig, err := getTLSConfig(hostPort, logger)
	if err != nil {
		logger.zl.Fatal("failed to build tls config", zap.Error(err))
	}

	stickyCacheSize := common.GetEnvOrDefaultInt(logger, "STICKY_CACHE_SIZE", 2048)
	worker.SetStickyWorkflowCacheSize(stickyCacheSize)

	startNamespaceWorker(logger, namespace, hostPort, tlsConfig, skipNamespaceCreation)

	// The workers are supposed to be long running process that should not exit.
	// Use select{} to block indefinitely for samples, you can quit by CMD+C.
	select {}
}

func createNamespaceIfNeeded(logger log.Logger, namespace string, hostPort string, tlsConfig *tls.Config) {
	logger.Info("Creating namespace", zap.String("namespace", namespace), zap.String("hostPort", hostPort))

	createNamespace := func() error {
		namespaceClient, err := client.NewNamespaceClient(client.Options{
			HostPort: hostPort,
			ConnectionOptions: client.ConnectionOptions{
				TLS: tlsConfig,
			},
		})
		if err != nil {
			logger.Error("failed to create Namespace Client", zap.Error(err))
			return err
		}

		defer namespaceClient.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		retention := 10 * time.Hour * 24
		err = namespaceClient.Register(ctx, &workflowservice.RegisterNamespaceRequest{
			Namespace:                        namespace,
			WorkflowExecutionRetentionPeriod: &retention,
		})

		if err == nil {
			logger.Info("Namespace created")
			time.Sleep(30 * time.Second)
			return nil
		}

		if _, ok := err.(*serviceerror.NamespaceAlreadyExists); ok {
			logger.Info("Namespace already exists")
			return nil
		}

		return err
	}

	for {
		err := createNamespace()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
}

func startNamespaceWorker(
	logger log.Logger,
	namespace string,
	hostPort string,
	tlsConfig *tls.Config,
	skipNamespaceCreation bool,
) {
	if !skipNamespaceCreation {
		createNamespaceIfNeeded(logger, namespace, hostPort, tlsConfig)
	}

	serviceClient, err := client.NewClient(client.Options{
		Namespace: namespace,
		HostPort:  hostPort,
		Logger:    logger,
		ConnectionOptions: client.ConnectionOptions{
			TLS: tlsConfig,
		},
		MetricsHandler: sdktally.NewMetricsHandler(newPrometheusScope(logger, prometheus.Configuration{
			ListenAddress: "0.0.0.0:9090",
			TimerType:     "histogram",
		})),
	})

	if err != nil {
		logger.(*ZapAdapter).zl.Fatal("failed to build temporal client", zap.Error(err))
	}

	workersString := common.GetEnvOrDefaultString(logger, "RUN_WORKERS", "bench,basic,basic-act")
	workers := strings.Split(workersString, ",")

	for _, workerName := range workers {
		var worker worker.Worker
		switch workerName {
		case "bench":
			worker = constructBenchWorker(context.Background(), serviceClient, logger, "temporal-bench")
		case "basic":
			worker = constructBasicWorker(context.Background(), serviceClient, logger, "temporal-basic")
		case "basic-act":
			worker = constructBasicActWorker(context.Background(), serviceClient, logger, "temporal-basic-act")
		default:
			panic(fmt.Sprintf("unknown worker %q", worker))
		}
		err = worker.Start()
		if err != nil {
			logger.(*ZapAdapter).zl.Fatal("Unable to start worker "+workerName, zap.Error(err))
		}
	}
}

func constructBenchWorker(ctx context.Context, serviceClient client.Client, logger log.Logger, taskQueue string) worker.Worker {
	w := worker.New(serviceClient, taskQueue, buildWorkerOptions(ctx, logger))
	w.RegisterWorkflowWithOptions(bench.Workflow, workflow.RegisterOptions{Name: "bench-workflow"})
	w.RegisterActivityWithOptions(bench.NewActivities(serviceClient), activity.RegisterOptions{Name: "bench-"})
	return w
}

func constructBasicWorker(ctx context.Context, serviceClient client.Client, logger log.Logger, taskQueue string) worker.Worker {
	w := worker.New(serviceClient, taskQueue, buildWorkerOptions(ctx, logger))
	w.RegisterWorkflowWithOptions(basic.Workflow, workflow.RegisterOptions{Name: "basic-workflow"})
	return w
}

func constructBasicActWorker(ctx context.Context, serviceClient client.Client, logger log.Logger, taskQueue string) worker.Worker {
	w := worker.New(serviceClient, taskQueue, buildWorkerOptions(ctx, logger))
	w.RegisterActivityWithOptions(basic.Activity, activity.RegisterOptions{Name: "basic-activity"})
	return w
}

func buildWorkerOptions(ctx context.Context, logger log.Logger) worker.Options {
	numDecisionPollers := common.GetEnvOrDefaultInt(logger, "NUM_DECISION_POLLERS", 50)
	logger.Info("Using env config for NUM_DECISION_POLLERS", zap.Int("NUM_DECISION_POLLERS", numDecisionPollers))

	workerOptions := worker.Options{
		BackgroundActivityContext:               ctx,
		MaxConcurrentWorkflowTaskPollers:        numDecisionPollers,
		MaxConcurrentActivityTaskPollers:        8 * numDecisionPollers,
		MaxConcurrentWorkflowTaskExecutionSize:  256,
		MaxConcurrentLocalActivityExecutionSize: 256,
		MaxConcurrentActivityExecutionSize:      256,
	}

	return workerOptions
}

func getTLSConfig(hostPort string, logger log.Logger) (*tls.Config, error) {
	host, _, parseErr := net.SplitHostPort(hostPort)
	if parseErr != nil {
		return nil, fmt.Errorf("unable to parse hostport properly: %+v", parseErr)
	}

	caCertData := common.GetEnvOrDefaultString(logger, "TLS_CA_CERT_DATA", "")
	clientCertData := common.GetEnvOrDefaultString(logger, "TLS_CLIENT_CERT_DATA", "")
	clientCertPrivateKeyData := common.GetEnvOrDefaultString(logger, "TLS_CLIENT_CERT_PRIVATE_KEY_DATA", "")
	caCertFile := common.GetEnvOrDefaultString(logger, "TLS_CA_CERT_FILE", "")
	clientCertFile := common.GetEnvOrDefaultString(logger, "TLS_CLIENT_CERT_FILE", "")
	clientCertPrivateKeyFile := common.GetEnvOrDefaultString(logger, "TLS_CLIENT_CERT_PRIVATE_KEY_FILE", "")
	enableHostVerification := common.GetEnvOrDefaultBool(logger, "TLS_ENABLE_HOST_VERIFICATION", false)

	caBytes, err := getTLSBytes(caCertFile, caCertData)
	if err != nil {
		return nil, err
	}

	certBytes, err := getTLSBytes(clientCertFile, clientCertData)
	if err != nil {
		return nil, err
	}

	keyBytes, err := getTLSBytes(clientCertPrivateKeyFile, clientCertPrivateKeyData)
	if err != nil {
		return nil, err
	}

	var cert *tls.Certificate
	var caPool *x509.CertPool

	if len(certBytes) > 0 {
		clientCert, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return nil, err
		}
		cert = &clientCert
	}

	if len(caBytes) > 0 {
		caPool = x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caBytes) {
			return nil, errors.New("unknown failure constructing cert pool for ca")
		}
	}

	// If we are given arguments to verify either server or client, configure TLS
	if caPool != nil || cert != nil {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: !enableHostVerification,
			ServerName:         host,
		}
		if caPool != nil {
			tlsConfig.RootCAs = caPool
		}
		if cert != nil {
			tlsConfig.Certificates = []tls.Certificate{*cert}
		}

		return tlsConfig, nil
	}

	return nil, nil

}

func getTLSBytes(certFile string, certData string) ([]byte, error) {
	var bytes []byte
	var err error

	if certFile != "" && certData != "" {
		return nil, errors.New("cannot specify both file and Base-64 encoded version of same field")
	}

	if certFile != "" {
		bytes, err = ioutil.ReadFile(certFile)
		if err != nil {
			return nil, err
		}
	} else if certData != "" {
		bytes, err = base64.StdEncoding.DecodeString(certData)
		if err != nil {
			return nil, err
		}
	}

	return bytes, err
}
