import * as docker from "@pulumi/docker"
import * as pulumi from "@pulumi/pulumi";
import * as random from "@pulumi/random"
import * as kubernetes from "@pulumi/kubernetes";

import * as authorization from "@pulumi/azure-native/authorization";
import * as containerregistry from "@pulumi/azure-native/containerregistry";

export interface ClusterArgs {
    kubeConfig: pulumi.Input<string>;
    principalId: pulumi.Input<string>;
}

export interface BenchArgs {
    resourceGroupName: pulumi.Input<string>;
    cluster: ClusterArgs;
    temporalFrontend: pulumi.Input<string>;
}

export class Bench extends pulumi.ComponentResource {
    constructor(name: string, args: BenchArgs) {
        super("my:example:Bench", name, args, undefined);

        const registry = new containerregistry.Registry("registry", {
            resourceGroupName: args.resourceGroupName,
            registryName: pulumi.output(args.resourceGroupName).apply(rg => rg.replace("-", "")),
            sku: {
                name: "Basic",
            },
            adminUserEnabled: true,
        }, { parent: this });
        
        const credentials = pulumi.all([args.resourceGroupName, registry.name]).apply(
            ([resourceGroupName, registryName]) => containerregistry.listRegistryCredentials({
                resourceGroupName: resourceGroupName,
                registryName: registryName,
        }));
        const adminUsername = credentials.apply(credentials => credentials.username!);
        const adminPassword = credentials.apply(credentials => credentials.passwords![0].value!);

        const clientConfig = pulumi.output(authorization.getClientConfig());

        const roleName = new random.RandomUuid("role-name", undefined, { parent: this });        
        new authorization.RoleAssignment("access-from-cluster", {
            principalId: args.cluster.principalId,
            roleAssignmentName: roleName.result,
            roleDefinitionId: pulumi.interpolate`/subscriptions/${clientConfig.subscriptionId}/providers/Microsoft.Authorization/roleDefinitions/7f951dda-4ed3-4680-a7ca-43fe172d538d`,
            scope: registry.id,
        }, { parent: this });
        
        const provider = new kubernetes.Provider("k8s-provider", {
            kubeconfig: args.cluster.kubeConfig,
            suppressDeprecationWarnings: true,
        }, { parent: this });
        
        const k8sOptions = { provider: provider, parent: this };

        const benchImageName = "temporal-bench-go";
        const benchImage = new docker.Image(benchImageName, {
            imageName: pulumi.interpolate`${registry.loginServer}/${benchImageName}`,
            build: { context: "../" },
            registry: {
                server: registry.loginServer,
                username: adminUsername,
                password: adminPassword,
            },
        }, { parent: this });

        const benchChart = new kubernetes.helm.v3.Chart("bench", {
            path: "../helm-chart",
            values: {
                image: {
                    repository: benchImage.imageName.apply(v => v.split(":")[0]),
                    tag: benchImage.imageName.apply(v => v.split(":")[1]),
                },
                tests: {
                    frontendAddress: args.temporalFrontend,
                    namespaceName: "default",
                }
            },
        }, k8sOptions);
        
        this.registerOutputs();
    }
}
