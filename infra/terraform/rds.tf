# ==============================================================================
# RDS POSTGRESQL MULTI-AZ CON GESTIÓN DE SECRETOS AUTOMATIZADA
# ==============================================================================

resource "aws_db_subnet_group" "rds" {
  name        = "${var.environment}-rds-subnet-group"
  subnet_ids  = aws_subnet.private_db[*].id
  description = "Subnets aisladas para PostgreSQL"

  tags = {
    Name = "${var.environment}-rds-subnet-group"
  }
}

# Generación aleatoria de contraseña de base de datos
resource "random_password" "db_password" {
  length           = 32
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# Almacenamiento seguro en AWS Secrets Manager (Sin secretos en texto plano)
resource "aws_secretsmanager_secret" "db_credentials" {
  name                    = "${var.environment}/platform/postgres"
  recovery_window_in_days = 7
  description             = "Credenciales maestras para RDS PostgreSQL gestionadas por Terraform"
}

resource "aws_secretsmanager_secret_version" "db_credentials" {
  secret_id = aws_secretsmanager_secret.db_credentials.id
  secret_string = jsonencode({
    engine   = "postgres"
    host     = aws_db_instance.postgres.address
    port     = 5432
    username = "platform_admin"
    password = random_password.db_password.result
    database = "platform_db"
  })
}

resource "aws_db_instance" "postgres" {
  identifier           = "${var.environment}-platform-postgres"
  allocated_storage    = 50
  max_allocated_storage = 500
  storage_type         = "gp3"
  engine               = "postgres"
  engine_version       = "16.3"
  instance_class       = var.rds_instance_class
  db_name              = "platform_db"
  username             = "platform_admin"
  password             = random_password.db_password.result

  db_subnet_group_name   = aws_db_subnet_group.rds.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  multi_az               = var.environment == "prod" ? true : false
  publicly_accessible    = false
  storage_encrypted      = true
  skip_final_snapshot    = var.environment == "prod" ? false : true
  deletion_protection    = var.environment == "prod" ? true : false

  backup_retention_period = 14
  backup_window           = "03:00-04:00"
  maintenance_window      = "Sun:04:30-Sun:05:30"

  tags = {
    Name = "${var.environment}-platform-postgres"
  }
}
