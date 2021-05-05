import * as pulumi from "@pulumi/pulumi";
import * as kubernetes from "@pulumi/kubernetes";

export interface CassandraArgs {
    type: "cassandra";
}

export interface ClusterArgs {
    kubeConfig: pulumi.Input<string>;
}

export interface ServerArgs {
    numHistoryShards?: number;
}

export interface TemporalArgs {
    version: string;
    storage: CassandraArgs;
    cluster: ClusterArgs;
    server: ServerArgs;
    visibility: "default" | "elasticsearch";
}

export class Temporal extends pulumi.ComponentResource {
    public webEndpoint: pulumi.Output<any>;
    public frontendAddress: pulumi.Output<string>;

    constructor(name: string, args: TemporalArgs) {
        super("my:example:Temporal", name, args, undefined);

        const provider = new kubernetes.Provider("k8s-provider", {
            kubeconfig: args.cluster.kubeConfig,
            suppressDeprecationWarnings: true,
        }, { parent: this });
        
        const k8sOptions = { provider: provider, parent: this };
        
        const chartValues: any = {                        
            web: {
                service: {
                    type: "LoadBalancer",
                }
            },
            server: {
                config: {
                    persistence: {
                        default: {
                            cassandra: {
                                hosts: ["cass-cassandra.default.svc.cluster.local"],
                                port: 9042,
                                keyspace: "temporal" + args.server.numHistoryShards?.toString(),
                                user: "user",
                                password: "password",
                                existingSecret: "",
                                replicationFactor: 1,
                                consistency: {
                                    default: {
                                        consistency: "local_quorum",
                                        serialConsistency: "local_serial",
                                    },
                                },
                            },
                        },              
                        visibility: {
                            cassandra: {
                                hosts: ["cass-cassandra.default.svc.cluster.local"],
                                port: 9042,
                                keyspace: "temporal_visibility",
                                user: "user",
                                password: "password",
                                existingSecret: "",
                                replicationFactor: 1,
                                consistency: {
                                    default: {
                                        consistency: "local_quorum",
                                        serialConsistency: "local_serial",
                                    },
                                },
                            },
                        },
                    },
                },
            },
            kafka: { enabled: false },
            cassandra: { enabled: false },
            prometheus: { enabled: false },
            elasticsearch: {
                enabled: false,
                external: true,
                host: "elasticsearch-master-headless",
                port: "9200",
                version: "v7",
                scheme: "http",
                logLevel: "error"
            },
            grafana: { enabled: false },
        };

        if (args.server.numHistoryShards) {
            chartValues.server.config.numHistoryShards = args.server.numHistoryShards;
        }

        const chart = new kubernetes.helm.v3.Chart("helm", {
            path: "../../../helm-charts", // adjust this to your location of https://github.com/temporalio/helm-charts
            values: chartValues,
        }, k8sOptions);
        
        const address = chart.getResourceProperty("v1/Service", "helm-temporal-web", "status").loadBalancer.ingress[0].ip;
        this.webEndpoint = pulumi.unsecret(pulumi.interpolate`http://${address}:8088`);

        const frontendIp = chart.getResourceProperty("v1/Service", "helm-temporal-frontend", "spec").clusterIP;
        this.frontendAddress = pulumi.interpolate`${frontendIp}:7233`;
       
        this.registerOutputs();
    }
}
