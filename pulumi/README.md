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

The deployment is structured as three Pulumi projects:

1. `k8s` deploys an AKS cluster with three node pools and third-party products to it (Cassandra, Elasticsearch, Prometheus, Grafana).
2. `temporal` deploys Temporal services.
3. `bench` deploys the Maru benchmark worker.
