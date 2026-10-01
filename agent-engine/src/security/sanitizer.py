"""Módulo de Sanitización y Prevención de Prompt Injection.

Implementa filtros heurísticos, análisis de delimitadores y normalización
antes de que cualquier input de usuario o evento externo sea procesado
por el Supervisor de LangGraph.
"""

from __future__ import annotations

import re
from typing import Final

# Expresiones regulares que detectan intentos típicos de sobreescritura de instrucciones
INJECTION_PATTERNS: Final[list[re.Pattern[str]]] = [
    re.compile(r"ignore\s+(all\s+)?(previous|prior)\s+instructions", re.IGNORECASE),
    re.compile(r"system\s*:\s*", re.IGNORECASE),
    re.compile(r"<\s*\|\s*im_start\s*\|\s*>", re.IGNORECASE),
    re.compile(r"<\s*\|\s*im_end\s*\|\s*>", re.IGNORECASE),
    re.compile(r"\[INST\].*?\[/INST\]", re.IGNORECASE | re.DOTALL),
    re.compile(r"(disregard|override|forget)\s+(the\s+)?(rules|prompt)", re.IGNORECASE),
    re.compile(r"you\s+are\s+now\s+(DAN|unrestricted|in\s+developer\s+mode)", re.IGNORECASE),
]

MAX_INPUT_LENGTH: Final[int] = 4096


class SecurityViolationError(ValueError):
    """Excepción lanzada cuando se detecta un posible vector de ataque."""
    pass


def sanitize_text(text: str) -> str:
    """Normaliza texto y valida que no exceda las cuotas de seguridad."""
    if not isinstance(text, str):
        raise SecurityViolationError("El input debe ser una cadena de texto.")
        
    cleaned = text.strip()
    if len(cleaned) > MAX_INPUT_LENGTH:
        raise SecurityViolationError(
            f"Longitud de texto excede el límite permitido de {MAX_INPUT_LENGTH} caracteres."
        )

    # Detección de patrones conocidos de inyección
    for pattern in INJECTION_PATTERNS:
        if pattern.search(cleaned):
            raise SecurityViolationError(
                f"Patrón de inyección no permitido detectado: '{pattern.pattern}'"
            )

    return cleaned
