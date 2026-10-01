terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.50"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.30"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }

  # Configuración recomendada para Backend Remoto S3 con bloqueo de estado DynamoDB
  # backend "s3" {
  #   bucket         = "platform-terraform-state-prod"
  #   key            = "hybrid-architecture/terraform.tfstate"
  #   region         = "us-east-1"
  #   dynamodb_table = "platform-terraform-locks"
  #   encrypt        = true
  # }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "Hybrid-AI-Platform"
      ManagedBy   = "Terraform"
      Environment = var.environment
    }
  }
}
