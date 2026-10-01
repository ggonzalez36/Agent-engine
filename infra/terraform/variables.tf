variable "aws_region" {
  type        = string
  description = "Región de AWS donde se desplegará la infraestructura"
  default     = "us-east-1"
}

variable "environment" {
  type        = string
  description = "Ambiente de despliegue (dev, staging, prod)"
  default     = "prod"
}

variable "vpc_cidr" {
  type        = string
  description = "Bloque CIDR principal de la VPC"
  default     = "10.0.0.0/16"
}

variable "cluster_name" {
  type        = string
  description = "Nombre del cluster de Kubernetes EKS"
  default     = "hybrid-platform-eks"
}

variable "rds_instance_class" {
  type        = string
  description = "Tipo de instancia computacional para PostgreSQL RDS"
  default     = "db.r6g.large"
}

variable "redis_node_type" {
  type        = string
  description = "Tipo de nodo para el cluster de Redis ElastiCache"
  default     = "cache.m6g.large"
}
