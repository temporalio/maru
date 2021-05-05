import * as pulumi from "@pulumi/pulumi";
import * as random from "@pulumi/random"
import * as kubernetes from "@pulumi/kubernetes";

export interface CassandraArgs {
    type: "cassandra";
    clusterSize: number;
}

export interface MySqlArgs {
    type: "mysql";
    hostName: pulumi.Input<string>;
    login: pulumi.Input<string>;
    password: pulumi.Input<string>;
}

export interface ClusterArgs {
    kubeConfig: pulumi.Input<string>;
    principalId: pulumi.Input<string>;
}

export interface ServerArgs {
    numHistoryShards?: number;
}

export interface TemporalArgs {
    resourceGroupName: pulumi.Input<string>;
    version: string;
    storage: CassandraArgs | MySqlArgs;
    cluster: ClusterArgs;
    server: ServerArgs;
    visibility: "default" | "elasticsearch";
}

export class Temporal extends pulumi.ComponentResource {
    public webEndpoint: pulumi.Output<any>;
    public grafanaPassword: pulumi.Output<any>;
    public frontendAddress: pulumi.Output<string>;

    constructor(name: string, args: TemporalArgs) {
        super("my:example:Temporal", name, args, undefined);

        const provider = new kubernetes.Provider("k8s-provider", {
            kubeconfig: args.cluster.kubeConfig,
            suppressDeprecationWarnings: true,
        }, { parent: this });
        
        const k8sOptions = { provider: provider, parent: this };
        
        if (args.storage.type == "mysql") {
            const temporalDefaultStorePassword = new kubernetes.core.v1.Secret("temporal-default-store", {
                metadata: {
                    name: "temporal-default-store",
                    namespace: "default",
                    labels: {
                        "app.kubernetes.io/name": "temporal",
                    }
                },
                type: "Opaque",
                data: {
                    password: pulumi.output(args.storage.password).apply(pwd => Buffer.from(pwd, "utf-8").toString("base64")),
                },
            }, k8sOptions);
        }

        const randomGrafanaPwd = new random.RandomPassword("granfana-password", {
            length: 12,
        }, { parent: this }).result;

        const chartValues: any = {                        
            grafana: {
                adminPassword: randomGrafanaPwd, // by default, Grafana's Helm chart would generate a new password on any deployment
            },
            web: {
                service: {
                    type: "LoadBalancer",
                }
            },
            server: {
                config: {                    
                },
                nodeSelector: {
                    agentpool: "agentpool",
                },
            },
            kafka: { enabled: false },
        };

        if (args.visibility == "default") {
            chartValues.elasticsearch = { enabled: false };
        }

        if (args.server.numHistoryShards) {
            chartValues.server.config.numHistoryShards = args.server.numHistoryShards;
        }

        if (args.storage.type == "mysql") {
            chartValues.server.config.persistence = {
                default: {
                    driver: "sql",
                    sql: {
                        driver: "mysql",
                        host: args.storage.hostName,
                        port: 3306,
                        database: "temporal",
                        user: args.storage.login,
                        password: args.storage.password,
                        maxConns: 20,
                        maxConnLifetime: "1h",
                    },
                },
                visibility: {
                    driver: "sql",
                    sql: {
                        driver: "mysql",
                        host: args.storage.hostName,
                        port: 3306,
                        database: "temporal_visibility",
                        user: args.storage.login,
                        password: args.storage.password,
                        maxConns: 20,
                        maxConnLifetime: "1h",
                    },
                },
            };
            chartValues.cassandra = {
                enabled: false,
            };
        } else if (args.storage.type === "cassandra") {
            chartValues.cassandra = {
                config: {
                    cluster_size: args.storage.clusterSize,
                },
            };
        }
        
        const chart = new kubernetes.helm.v3.Chart("helm", {
            path: "../../helm-charts",            
            values: chartValues,
        }, k8sOptions);
        
        const address = chart.getResourceProperty("v1/Service", "helm-temporal-web", "status").loadBalancer.ingress[0].ip;
        this.webEndpoint = pulumi.unsecret(pulumi.interpolate`http://${address}:8088`);

        const frontendIp = chart.getResourceProperty("v1/Service", "helm-temporal-frontend", "spec").clusterIP;
        this.frontendAddress = pulumi.interpolate`${frontendIp}:7233`;

        const grafanaCreds = pulumi.unsecret(chart.getResourceProperty("v1/Secret", "default/helm-grafana", "data"));
        this.grafanaPassword = grafanaCreds["admin-password"].apply(v => Buffer.from(v, "base64").toString());
        
        this.registerOutputs();
    }
}
