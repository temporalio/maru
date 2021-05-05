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

import * as authorization from "@pulumi/azure-native/authorization";
import * as containerregistry from "@pulumi/azure-native/containerregistry";
import * as resources from "@pulumi/azure-native/resources";
import * as pulumi from "@pulumi/pulumi";
import * as kubernetes from "@pulumi/kubernetes";
import * as random from "@pulumi/random";

import { AksCluster } from "./cluster";

const resourceGroup = new resources.ResourceGroup("mikhail-rg");

// Create an AKS cluster with a default node pool and two extra node pools
const cluster = new AksCluster("aks", {
    resourceGroupName: resourceGroup.name,
    kubernetesVersion: "1.18.14",
    vmSize: "Standard_DS2_v2",
    vmCount: 3,
    nodePools: [{
        name: "casspool",
        target: "cassandra",
        vmSize: "Standard_DS4_v2",
        vmCount: 3,
    },{
        name: "elasticpool",
        target: "elastic",
        vmSize: "Standard_DS2_v2",
        vmCount: 3,
    }],
});

// Create a container registry for future deployments of the bench container
const registry = new containerregistry.Registry("registry", {
    resourceGroupName: resourceGroup.name,
    sku: {
        name: "Basic",
    },
    adminUserEnabled: true,
});

// Export the credentials to the registry so that other stacks could upload to it
const credentials = pulumi.all([resourceGroup.name, registry.name]).apply(
    ([resourceGroupName, registryName]) => containerregistry.listRegistryCredentials({
        resourceGroupName: resourceGroupName,
        registryName: registryName,
}));
export const registryLoginServer = registry.loginServer;
export const registryAdminUsername = credentials.apply(credentials => credentials.username!);
export const registryAdminPassword = credentials.apply(credentials => credentials.passwords![0].value!);

// Grant access to the registry from the cluster to pull containers
const clientConfig = pulumi.output(authorization.getClientConfig());
new authorization.RoleAssignment("access-from-cluster", {
    principalId: cluster.principalId,
    principalType: "ServicePrincipal",
    roleDefinitionId: pulumi.interpolate`/subscriptions/${clientConfig.subscriptionId}/providers/Microsoft.Authorization/roleDefinitions/7f951dda-4ed3-4680-a7ca-43fe172d538d`,
    scope: registry.id,
});

// Create an explicit Kubernetes provider to connect to the new cluster
const provider = new kubernetes.Provider("k8s-provider", {
    kubeconfig: cluster.kubeConfig,
    suppressDeprecationWarnings: true,
});
const k8sOptions = { provider: provider };

// Deploy Cassandra Helm chart
const cassandra = new kubernetes.helm.v3.Chart("cass", {
    fetchOpts: {
        repo: "https://charts.helm.sh/incubator",
    },
    chart: "cassandra",
    version: "0.14.3",
    values: {
        service: {
            type: "ClusterIP",
        },
        selector: {
            nodeSelector: {
                target: "cassandra",
            },
        },
        tolerations: [{
            key: "target",
            operator: "Equal",
            value: "cassandra",
            effect: "NoSchedule",
        }],
    },
}, k8sOptions);

// Deploy Elasticsearch Helm chart
const elastic = new kubernetes.helm.v3.Chart("elastic", {
    fetchOpts: {
        repo: "https://helm.elastic.co",
    },
    chart: "elasticsearch",
    version: "7.12.0",
    values: {
        nodeSelector: {
            target: "elastic",
        },
        tolerations: [{
            key: "target",
            operator: "Equal",
            value: "elastic",
            effect: "NoSchedule",
        }],
    },
}, k8sOptions);

// Deploy Prometheus Helm chart
const prometheus = new kubernetes.helm.v3.Chart("prometheus", {
    fetchOpts: {
        repo: "https://prometheus-community.github.io/helm-charts",
    },
    chart: "prometheus",
    version: "11.0.4",
}, k8sOptions);

const randomGrafanaPwd = new random.RandomPassword("granfana-password", {
    length: 12,
}).result;

// Deploy Grafana Helm chart
const grafana = new kubernetes.helm.v3.Chart("grafana", {
    fetchOpts: {
        repo: "https://grafana.github.io/helm-charts",
    },
    chart: "grafana",
    version: "5.0.10",
    values: {
        replicas: 1,
        adminPassword: randomGrafanaPwd, // by default, Grafana's Helm chart would generate a new password on any deployment
        testFramework: {
            enabled: false,
        },
        rbac: {
            create: false,
            pspEnabled: false,
            namespaced: true,
        },
        dashboardProviders: {
            "dashboardproviders.yaml": {
                apiVersion: 1,
                providers: [{
                    name: "default",
                    orgId: 1,
                    folder: '',
                    type: "file",
                    disableDeletion: false,
                    editable: true,
                    options: {
                        path: "/var/lib/grafana/dashboards/default",
                    },
                }],
            },
        },
        datasources: {
            "datasources.yaml": {
                apiVersion: 1,
                datasources: [{
                    name: "TemporalMetrics",
                    type: "prometheus",
                    url: "http://prometheus-server",
                    access: "proxy",
                    isDefault: true,
                }],
            },
        },
        dashboards: {
            default: {
                "frontend-github": {
                    url: "https://raw.githubusercontent.com/temporalio/temporal-dashboards/master/dashboards/frontend.json",
                    datasource: "TemporalMetrics",
                },
                "temporal-github": {
                    url: "https://raw.githubusercontent.com/temporalio/temporal-dashboards/master/dashboards/temporal.json",
                    datasource: "TemporalMetrics",
                },
                "history-github": {
                    url: "https://raw.githubusercontent.com/temporalio/temporal-dashboards/master/dashboards/history.json",
                    datasource: "TemporalMetrics",
                },
                "matching-github": {
                    url: "https://raw.githubusercontent.com/temporalio/temporal-dashboards/master/dashboards/matching.json",
                    datasource: "TemporalMetrics",
                },
                "clusteroverview-github": {
                    url: "https://raw.githubusercontent.com/temporalio/temporal-dashboards/master/dashboards/10000.json",
                    datasource: "TemporalMetrics",
                },
                "common-github": {
                    url: "https://raw.githubusercontent.com/temporalio/temporal-dashboards/master/dashboards/common.json",
                    datasource: "TemporalMetrics",
                },
            },
        },
    },
}, k8sOptions);

const grafanaCreds = pulumi.unsecret(grafana.getResourceProperty("v1/Secret", "default/grafana", "data"));
export const grafanaPassword = grafanaCreds["admin-password"].apply(v => Buffer.from(v, "base64").toString());
export const kubeconfig = cluster.kubeConfig;
