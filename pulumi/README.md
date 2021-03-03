[![Deploy](https://get.pulumi.com/new/button.svg)](https://app.pulumi.com/new)

# Deploy Temporal Bench to Azure Kubernetes Service (AKS)

The example uses [Pulumi](https://www.pulumi.com) to deploy several Azure and Temporal components to Azure managed services:

- New Azure Resource Group
- Azure AD Service Principal and SSH Key
- Azure Kubernetes Service (AKS) cluster
- Azure Container Registry with a role assignment for the AKS SP
- Docker image with the bench workflow implementation
- Temporal Server and all its dependencies using the Temporal Helm chart
- Temporal Bench and all its dependencies using the Bench Helm chart
- Basic workflow as a test target

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

1.  Run `pulumi up` to preview and deploy changes.


1.  Follow the instructions in the [top-level README](../README.md) to run the benchmark.
