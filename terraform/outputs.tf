# Outputs for Waterflow infrastructure

output "vpc_id" {
  description = "ID of the VPC"
  value       = module.networking.vpc_id
}

output "subnet_ids" {
  description = "IDs of the subnets"
  value       = module.networking.subnet_ids
}

output "cluster_endpoint" {
  description = "Endpoint for Kubernetes cluster"
  value       = module.kubernetes.cluster_endpoint
}

output "cluster_ca_certificate" {
  description = "CA certificate for Kubernetes cluster"
  value       = module.kubernetes.cluster_ca_certificate
  sensitive   = true
}

output "database_endpoint" {
  description = "Database endpoint"
  value       = module.database.database_endpoint
}

output "redis_endpoint" {
  description = "Redis endpoint"
  value       = module.redis.redis_endpoint
}

output "load_balancer_dns" {
  description = "DNS name of the load balancer"
  value       = module.load_balancer.load_balancer_dns
}

output "waterflow_service_url" {
  description = "URL for Waterflow service"
  value       = "http://${module.load_balancer.load_balancer_dns}"
}