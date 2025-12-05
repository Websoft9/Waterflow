# Development environment configuration
aws_region     = "us-east-1"
gcp_project    = "waterflow-dev"
gcp_region     = "us-central1"
azure_location = "East US"
environment    = "dev"
cluster_name   = "waterflow-dev-cluster"
node_count     = 2
machine_type   = "n1-standard-1"
vpc_cidr       = "10.0.0.0/16"
subnet_cidrs   = ["10.0.1.0/24", "10.0.2.0/24"]
tags = {
  Project     = "Waterflow"
  Environment = "dev"
  ManagedBy   = "Terraform"
  Owner       = "DevOps Team"
}