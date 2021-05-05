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

import * as pulumi from "@pulumi/pulumi";
import * as docker from "@pulumi/docker";
import * as kubernetes from "@pulumi/kubernetes";

const config = new pulumi.Config();

const k8s = new pulumi.StackReference(config.require("k8s-stack-ref"));
const temporal = new pulumi.StackReference(config.require("temporal-stack-ref"));

const provider = new kubernetes.Provider("k8s-provider", {
    kubeconfig: k8s.requireOutput("kubeconfig"),
    suppressDeprecationWarnings: true,
});

const k8sOptions = { provider: provider };

const benchImageName = "temporal-bench-go";
const loginServer = k8s.requireOutput("registryLoginServer");
const benchImage = new docker.Image(benchImageName, {
    imageName: pulumi.interpolate`${loginServer}/${benchImageName}`,
    build: { context: "../../worker/" },
    registry: {
        server: loginServer,
        username: k8s.requireOutput("registryAdminUsername"),
        password: k8s.requireOutput("registryAdminPassword"),
    },
});

const benchChart = new kubernetes.helm.v3.Chart("bench", {
    path: "../../helm-chart",
    values: {
        image: {
            repository: benchImage.imageName.apply(v => v.split(":")[0]),
            tag: benchImage.imageName.apply(v => v.split(":")[1]),
        },
        tests: {
            frontendAddress: temporal.requireOutput("frontendAddress"),
            namespaceName: "default",
        }
    },
}, k8sOptions);
