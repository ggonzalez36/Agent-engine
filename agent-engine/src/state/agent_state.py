"""Definición del estado global del agente (AgentState).

Principios inquebrantables de diseño:
1. Inmutabilidad y semántica Append-Only para el historial de mensajes, auditoría y errores.
2. Contratos tipados y serializables para checkpoints deterministas.
"""

from __future__ import annotations

import operator
from typing import Annotated, Any, Literal, TypedDict
from langchain_core.messages import BaseMessage
from langgraph.graph.message import add_messages
from pydantic import BaseModel, Field


class AuditEntry(TypedDict):
    step: str
    agent: str
    action: str
    timestamp: str
    details: dict[str, Any]


class RiskAssessment(BaseModel):
    score: float = Field(ge=0.0, le=1.0, description="Puntuación de riesgo normalizada entre 0 y 1")
    risk_level: Literal["LOW", "MEDIUM", "HIGH", "CRITICAL"]
    flags: list[str] = Field(default_factory=list)
    reasoning: str


class ComplianceCheck(BaseModel):
    is_compliant: bool
    sanctions_clear: bool
    fraud_detected: bool
    violations: list[str] = Field(default_factory=list)


class RouterDecision(BaseModel):
    """Salida estructurada requerida para el Supervisor de LangGraph."""
    next_node: Literal["risk_evaluator", "compliance_checker", "order_finalizer", "FINISH"] = Field(
        description="El siguiente agente especialista que debe activarse o FINISH si se completó el flujo."
    )
    reasoning: str = Field(description="Explicación técnica determinista de la decisión de enrutamiento.")
    confidence: float = Field(ge=0.0, le=1.0, description="Nivel de certeza del enrutamiento.")


class AgentState(TypedDict):
    """Estado compartido en estrella gobernado por el Supervisor.
    
    Toda colección mutable debe utilizar operadores reducer (add_messages, operator.add)
    para evitar mutaciones destructivas en memoria y garantizar trazabilidad auditable.
    """
    # Historial de conversación / ejecuciones (Append-Only)
    messages: Annotated[list[BaseMessage], add_messages]
    
    # Metadatos de correlación e identidad de dominio
    order_id: str
    customer_id: str
    correlation_id: str
    causation_id: str
    
    # Registro de fase actual
    current_phase: str
    
    # Evaluaciones intermedias de agentes especialistas
    risk_assessment: RiskAssessment | None
    compliance_check: ComplianceCheck | None
    
    # Bitácora inmutable de auditoría (Append-Only)
    audit_trail: Annotated[list[AuditEntry], operator.add]
    
    # Registro de errores tolerantes a fallos (Append-Only)
    errors: Annotated[list[str], operator.add]
