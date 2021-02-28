import * as resources from "@pulumi/azure-native/resources";
import * as pulumi from "@pulumi/pulumi";

import { resourceGroupName } from "./config";
import { AksCluster } from "./cluster";
import { Temporal } from "./temporal";
import { Bench } from "./bench";

const resourceGroup = new resources.ResourceGroup("resourceGroup", {
    resourceGroupName: resourceGroupName,
    tags: {
        Owner: "mikhail",
    },
});

const config = new pulumi.Config();

const cluster = new AksCluster("aks", {
    resourceGroupName: resourceGroup.name,
    kubernetesVersion: config.require("aks.version"),
    vmSize: config.require("aks.vmsize"),
    vmCount: config.requireNumber("aks.vmcount"),
});

const temporal = new Temporal("temporal", {
    resourceGroupName: resourceGroup.name,
    version: config.require("temporal.version"),
    storage: {
        type: "cassandra",
        clusterSize: config.requireNumber("cassandra.clustersize"),
    },
    cluster: cluster,
    visibility: config.require("temporal.visibility"),
});

const bench = new Bench("bench", {
    resourceGroupName: resourceGroup.name,
    cluster: cluster,
    temporalFrontend: temporal.frontendAddress,
});

//export const kubeconfig = cluster.kubeConfig;

export const grafanaPassword = temporal.grafanaPassword;
export const endpoints = {
    web: temporal.webEndpoint,
    grafana: "http://localhost:8081",
};
export const kubectlCommands = {
    frontendPortForward: "kubectl port-forward services/helm-temporal-frontend 7000:7233",
    grafanaPortForward: "kubectl port-forward services/helm-grafana 8081:80",
    elasticSearchPortForward: "kubectl port-forward services/elasticsearch-master 9200:9200",
    logs: "kubectl logs -l app.kubernetes.io/name=temporal-bench --follow",
};
