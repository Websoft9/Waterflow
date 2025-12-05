# Waterflow Infrastructure

This directory contains Terraform configurations for deploying Waterflow infrastructure across multiple cloud providers.

## Supported Providers

- **AWS**: ECS, EKS, RDS, ElastiCache
- **Google Cloud**: GKE, Cloud SQL, Memorystore
- **Azure**: AKS, Database for PostgreSQL, Cache for Redis

## Quick Start

### Prerequisites

- Terraform >= 1.0
- AWS CLI configured (for AWS deployments)
- Google Cloud SDK configured (for GCP deployments)
- Azure CLI configured (for Azure deployments)

### Initialize Terraform

```bash
cd terraform
terraform init
```

### Plan Deployment

```bash
terraform plan -var-file="environments/dev.tfvars"
```

### Apply Changes

```bash
terraform apply -var-file="environments/dev.tfvars"
```

## Directory Structure

```
terraform/
├── main.tf              # Main Terraform configuration
├── variables.tf         # Input variables
├── outputs.tf           # Output values
├── environments/        # Environment-specific configurations
│   ├── dev.tfvars
│   ├── staging.tfvars
│   └── prod.tfvars
├── modules/             # Reusable Terraform modules
│   ├── networking/
│   ├── kubernetes/
│   ├── database/
│   └── monitoring/
└── README.md
```

## Environment Configuration

Create environment-specific variable files in the `environments/` directory:

```hcl
# environments/dev.tfvars
aws_region     = "us-east-1"
gcp_project    = "my-waterflow-dev"
environment    = "dev"
node_count     = 2
machine_type   = "n1-standard-1"
```

## Modules

### Networking Module
- VPC and subnet creation
- Security groups and firewall rules
- Load balancer configuration

### Kubernetes Module
- Cluster provisioning
- Node pool configuration
- RBAC setup

### Database Module
- PostgreSQL instance setup
- High availability configuration
- Backup and recovery

### Monitoring Module
- Prometheus and Grafana setup
- Alert manager configuration
- Log aggregation

## Security

- All secrets are managed through cloud provider secret managers
- Network isolation with security groups/firewall rules
- Least privilege IAM roles
- Encryption at rest and in transit

## Cost Optimization

- Auto-scaling based on resource utilization
- Spot instances for non-production workloads
- Reserved instances for production
- Resource tagging for cost tracking

## Contributing

1. Follow Terraform best practices
2. Use modules for reusable components
3. Include proper documentation
4. Test changes in development environment first