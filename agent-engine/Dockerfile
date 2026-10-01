# ==============================================================================
# STAGE 1: Dependency Builder
# ==============================================================================
FROM python:3.11-slim AS builder

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /install

# Herramientas mínimas de compilación
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Entorno virtual aislado
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

COPY pyproject.toml ./
# Instalación de dependencias del proyecto en el venv
RUN pip install --upgrade pip setuptools wheel && \
    pip install .

# ==============================================================================
# STAGE 2: Secure Unprivileged Runtime (DevSecOps Baseline)
# ==============================================================================
FROM python:3.11-slim AS runner

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PATH="/opt/venv/bin:$PATH" \
    PYTHONPATH="/app"

# Instalación de dumb-init para gestión adecuada de señales PID 1 (SIGTERM/SIGINT)
RUN apt-get update && apt-get install -y --no-install-recommends dumb-init && \
    rm -rf /var/lib/apt/lists/*

# Creación de usuario y grupo no privilegiado (UID 10001) sin acceso a shell
RUN groupadd -g 10001 appgroup && \
    useradd -u 10001 -g appgroup -s /sbin/nologin -M -d /app appuser

WORKDIR /app

# Copia exclusiva del entorno virtual desde el builder
COPY --from=builder --chown=appuser:appgroup /opt/venv /opt/venv

# Copia del código fuente con permisos restringidos
COPY --chown=appuser:appgroup src/ /app/src/

# Forzado de ejecución bajo usuario no root
USER 10001:10001

ENTRYPOINT ["/usr/bin/dumb-init", "--", "python", "-m", "src.consumer.redis_consumer"]
