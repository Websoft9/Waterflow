# Waterflow Infrastructure as Code
# Terraform configuration for deploying Waterflow to cloud providers

terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 4.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }

  backend "s3" {
    # Configure backend for state management
    # bucket = "waterflow-terraform-state"
    # key    = "waterflow.tfstate"
    # region = "us-east-1"
  }
}

# Provider configurations
provider "aws" {
  region = var.aws_region
}

provider "google" {
  project = var.gcp_project
  region  = var.gcp_region
}

provider "azurerm" {
  features {}
}

# Variables
variable "aws_region" {
  description = "AWS region for deployment"
  type        = string
  default     = "us-east-1"
}

variable "gcp_project" {
  description = "GCP project ID"
  type        = string
  default     = ""
}

variable "gcp_region" {
  description = "GCP region for deployment"
  type        = string
  default     = "us-central1"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default = {
    Project     = "Waterflow"
    Environment = "dev"
    ManagedBy   = "Terraform"
  }
}

# Outputs
output "aws_ecs_cluster_name" {
  description = "AWS ECS cluster name"
  value       = module.aws_infrastructure.ecs_cluster_name
}

output "gcp_gke_cluster_name" {
  description = "GCP GKE cluster name"
  value       = module.gcp_infrastructure.gke_cluster_name
}

output "azure_aks_cluster_name" {
  description = "Azure AKS cluster name"
  value       = module.azure_infrastructure.aks_cluster_name
}