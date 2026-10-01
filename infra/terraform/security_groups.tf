# ==============================================================================
# SECURITY GROUPS (ZERO TRUST NETWORK SEGMENTATION)
# ==============================================================================

# 1. Security Group para Nodos EKS
resource "aws_security_group" "eks_nodes" {
  name        = "${var.environment}-eks-nodes-sg"
  description = "Reglas de trafico para los worker nodes de Kubernetes"
  vpc_id      = aws_vpc.main.id

  # Trafico interno entre pods y nodos
  ingress {
    description = "Comunicacion interna de nodos y pods"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    self        = true
  }

  # Salida habilitada hacia internet via NAT GW (requerido para APIs LLM y descargas)
  egress {
    description = "Salida general para llamadas HTTPS (OpenAI, AWS APIs, etc.)"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.environment}-eks-nodes-sg"
  }
}

# 2. Security Group para RDS PostgreSQL
resource "aws_security_group" "rds" {
  name        = "${var.environment}-rds-postgres-sg"
  description = "Acceso restringido exclusivamente desde el cluster EKS"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "Acceso PostgreSQL 5432 solo desde nodos EKS"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_nodes.id]
  }

  egress {
    description = "Sin trafico saliente permitido"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.environment}-rds-sg"
  }
}

# 3. Security Group para Redis ElastiCache
resource "aws_security_group" "redis" {
  name        = "${var.environment}-redis-sg"
  description = "Acceso a Redis Pub/Sub restringido desde los worker nodes"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "Redis port 6379 solo desde nodos EKS"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_nodes.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.environment}-redis-sg"
  }
}
