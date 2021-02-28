[![Deploy](https://get.pulumi.com/new/button.svg)](https://app.pulumi.com/new)

# Deploy a Temporal application to Azure Kubernetes Service (AKS)

Starting point for building an instance of Temporal Server and Workflows in AKS.

The example uses [Pulumi](https://www.pulumi.com) to deploy several Temporal components to Azure managed services:

- New Azure Resource Group
- Azure MySQL Database for storage
- Azure AD Service Principal and SSH Key
- Azure Kubernetes Service (AKS) cluster
- Azure Container Registry with a role assignment for the AKS SP
- Docker image with a sample workflow implementation
- Temporal Server as a deployment and a ClusterIP service
- Temporal Web Console as a deployment and a ClusterIP service
- Temporal Worker and HTTP server to start workflows

## Running the App

1.  Create a new stack:

    ```
    $ pulumi stack init dev
    ```

1.  Login to Azure CLI (you will be prompted to do this during deployment if you forget this step):

    ```
    $ az login
    ```

1.  Restore NPM dependencies:

    ```
    $ npm install
    ```

1.  Run `pulumi up` to preview and deploy changes:

    ```
    $ pulumi up
         Type                                                        Name                    Status      
    +   pulumi:pulumi:Stack                                         tempora-azure-aks-dev   created     
    +   ├─ my:example:MySql                                         mysql                   created     
    +   │  ├─ azure-nextgen:dbformysql/latest:Server                mysql                   created     
    +   │  └─ azure-nextgen:dbformysql/latest:FirewallRule          mysql-allow-azure       created     
    +   ├─ my:example:Temporal                                      temporal                created     
    +   │  ├─ docker:image:Image                                    temporal-worker         created     
    +   │  ├─ azure-nextgen:containerregistry/latest:Registry       registry                created     
    +   │  ├─ azure-nextgen:authorization/latest:RoleAssignment     access-from-cluster     created     
    +   │  ├─ pulumi:providers:kubernetes                           k8s-provider            created     
    +   │  ├─ kubernetes:core/v1:Namespace                          temporal-ns             created     
    +   │  ├─ kubernetes:core/v1:Secret                             temporal-default-store  created     
    +   │  ├─ kubernetes:apps/v1:Deployment                         temporal-web            created     
    +   │  ├─ kubernetes:apps/v1:Deployment                         workflow-app            created     
    +   │  ├─ kubernetes:apps/v1:Deployment                         temporal-worker         created     
    +   │  ├─ kubernetes:core/v1:Service                            temporal-web            created     
    +   │  ├─ kubernetes:core/v1:Service                            temporal-worker         created     
    +   │  └─ kubernetes:core/v1:Service                            workflow-app            created     
    +   ├─ my:example:AksCluster                                    aks                     created     
    +   │  ├─ random:index:RandomPassword                           password                created     
    +   │  ├─ tls:index:PrivateKey                                  ssh-key                 created     
    +   │  ├─ azuread:index:ServicePrincipal                        aksSp                   created     
    +   │  ├─ azuread:index:ServicePrincipalPassword                aksSpPassword           created     
    +   │  └─ azure-nextgen:containerservice/latest:ManagedCluster  managedCluster          created     
    +   ├─ azuread:index:Application                                aks                     created     
    +   ├─ random:index:RandomString                                resourcegroup-name      created     
    +   ├─ random:index:RandomPassword                              mysql-password          created     
    +   └─ azure-nextgen:resources/latest:ResourceGroup             resourceGroup           created     
    
    Outputs:
        starterEndpoint: "http://21.55.177.186:8080/async?name="
        webEndpoint    : "http://52.136.6.198:8088"

    Resources:
        + 27 created

    Duration: 6m46s
    ```

1.  Start a workflow:

    ```
    $ pulumi stack output starterEndpoint
    http://21.55.177.186:8080/async?name=
    $ curl $(pulumi stack output starterEndpoint)World
    Started workflow ID=World, RunID=b4f6db00-bb2f-498b-b620-caad81c91a81% 
    ```

1. Navigate to Temporal Web console:

    ```
    $ pulumi stack output webEndpoint
    http://52.136.6.198:8088 # Open in your browser
    ```
