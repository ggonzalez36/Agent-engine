# ==============================================================================
# ELASTICACHE REDIS (PUB/SUB & CACHE BROKER)
# ==============================================================================

resource "aws_elasticache_subnet_group" "redis" {
  name       = "${var.environment}-redis-subnet-group"
  subnet_ids = aws_subnet.private_db[*].id

  tags = {
    Name = "${var.environment}-redis-subnet-group"
  }
}

resource "random_password" "redis_auth_token" {
  length  = 32
  special = false
}

resource "aws_secretsmanager_secret" "redis_secret" {
  name                    = "${var.environment}/platform/redis"
  recovery_window_in_days = 7
  description             = "Auth Token y Endpoints para Redis ElastiCache"
}

resource "aws_secretsmanager_secret_version" "redis_secret" {
  secret_id = aws_secretsmanager_secret.redis_secret.id
  secret_string = jsonencode({
    primary_endpoint = aws_elasticache_replication_group.redis.primary_endpoint_address
    port             = 6379
    auth_token       = random_password.redis_auth_token.result
  })
}

resource "aws_elasticache_replication_group" "redis" {
  replication_group_id       = "${var.environment}-platform-redis"
  description                = "Cluster Redis para Outbox Relay Pub/Sub y Telemetria SSE"
  node_type                  = var.redis_node_type
  num_cache_clusters         = 2
  parameter_group_name       = "default.redis7"
  port                       = 6379
  subnet_group_name          = aws_elasticache_subnet_group.redis.name
  security_group_ids         = [aws_security_group.redis.id]
  automatic_failover_enabled = true
  multi_az_enabled           = true

  # Cifrado en reposo y en transito (Mandatorio por seguridad)
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                 = random_password.redis_auth_token.result

  tags = {
    Name = "${var.environment}-redis-cluster"
  }
}
