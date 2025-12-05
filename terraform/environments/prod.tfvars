# Production environment configuration
aws_region     = "us-east-1"
gcp_project    = "waterflow-prod"
gcp_region     = "us-central1"
azure_location = "East US"
environment    = "prod"
cluster_name   = "waterflow-prod-cluster"
node_count     = 5
machine_type   = "n1-standard-4"
vpc_cidr       = "10.0.0.0/16"
subnet_cidrs   = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
database_instance_class = "db.r5.large"
redis_node_type         = "cache.r5.large"
tags = {
  Project     = "Waterflow"
  Environment = "prod"
  ManagedBy   = "Terraform"
  Owner       = "DevOps Team"
  Backup      = "daily"
  Monitoring  = "enabled"
}