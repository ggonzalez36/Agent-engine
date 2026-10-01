# ==============================================================================
# IAM ROLES FOR SERVICE ACCOUNTS (IRSA - ZERO TRUST POD IDENTITY)
# ==============================================================================

# 1. Configuración de OIDC Provider para el Cluster EKS
data "tls_certificate" "eks" {
  url = aws_eks_cluster.main.identity[0].oidc[0].issuer
}

resource "aws_iam_openid_connect_provider" "eks" {
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.eks.certificates[0].sha1_fingerprint]
  url             = aws_eks_cluster.main.identity[0].oidc[0].issuer
}

# 2. IRSA para Backend Transaccional en Go (Permiso exclusivo a RDS + Redis)
data "aws_iam_policy_document" "backend_assume_role" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    effect  = "Allow"

    condition {
      test     = "StringEquals"
      variable = "${replace(aws_iam_openid_connect_provider.eks.url, "https://", "")}:sub"
      values   = ["system:serviceaccount:production:backend-service-account"]
    }

    principals {
      identifiers = [aws_iam_openid_connect_provider.eks.arn]
      type        = "Federated"
    }
  }
}

resource "aws_iam_role" "backend_sa" {
  name               = "${var.environment}-backend-sa-role"
  assume_role_policy = data.aws_iam_policy_document.backend_assume_role.json
}

resource "aws_iam_policy" "backend_secrets" {
  name        = "${var.environment}-backend-secrets-policy"
  description = "Permite acceso a credenciales de RDS y Redis para el Backend en Go"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = [
          aws_secretsmanager_secret.db_credentials.arn,
          aws_secretsmanager_secret.redis_secret.arn
        ]
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "backend_attach" {
  role       = aws_iam_role.backend_sa.name
  policy_arn = aws_iam_policy.backend_secrets.arn
}

# 3. IRSA para Motor Agéntico LangGraph (ZERO TRUST: CERO ACCESO A RDS)
data "aws_iam_policy_document" "agent_assume_role" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    effect  = "Allow"

    condition {
      test     = "StringEquals"
      variable = "${replace(aws_iam_openid_connect_provider.eks.url, "https://", "")}:sub"
      values   = ["system:serviceaccount:production:agent-engine-service-account"]
    }

    principals {
      identifiers = [aws_iam_openid_connect_provider.eks.arn]
      type        = "Federated"
    }
  }
}

resource "aws_iam_role" "agent_sa" {
  name               = "${var.environment}-agent-sa-role"
  assume_role_policy = data.aws_iam_policy_document.agent_assume_role.json
}

# Secret para llaves LLM
resource "aws_secretsmanager_secret" "llm_keys" {
  name                    = "${var.environment}/platform/llm-keys"
  recovery_window_in_days = 7
  description             = "API Keys para OpenAI y proveedores LLM"
}

resource "aws_iam_policy" "agent_secrets" {
  name        = "${var.environment}-agent-secrets-policy"
  description = "Permite acceso a LLM Keys y Redis. CERO permisos a RDS Postgres."

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = [
          aws_secretsmanager_secret.llm_keys.arn,
          aws_secretsmanager_secret.redis_secret.arn
        ]
      }
      # NOTA DE SEGURIDAD ZERO TRUST:
      # aws_secretsmanager_secret.db_credentials NO está incluido en esta política.
      # El Pod del motor agéntico tiene prohibido por IAM obtener las credenciales de la DB.
    ]
  })
}

resource "aws_iam_role_policy_attachment" "agent_attach" {
  role       = aws_iam_role.agent_sa.name
  policy_arn = aws_iam_policy.agent_secrets.arn
}
