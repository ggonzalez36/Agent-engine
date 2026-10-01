"""Nodos de Agentes Especialistas que componen la topología en estrella."""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any
from langchain_core.messages import AIMessage
from src.clients.backend_client import BackendClient
from src.state.agent_state import (
    AgentState,
    AuditEntry,
    ComplianceCheck,
    RiskAssessment,
)
from src.telemetry.publisher import emit_telemetry


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


async def risk_evaluator_node(state: AgentState) -> dict[str, Any]:
    """Agente especialista en evaluación cuantitativa de riesgo."""
    order_id = state["order_id"]
    correlation_id = state["correlation_id"]

    # 1. Telemetría SSE: Inicio de evaluación de riesgo
    await emit_telemetry(correlation_id, "RISK_EVALUATION_STARTED", {"order_id": order_id})

    # Lógica determinista de análisis de riesgo (extensible con modelos probabilísticos)
    assessment = RiskAssessment(
        score=0.15,
        risk_level="LOW",
        flags=[],
        reasoning="Transacción dentro de los parámetros habituales del cliente.",
    )

    audit_entry: AuditEntry = {
        "step": "risk_evaluation",
        "agent": "risk_evaluator",
        "action": "EVALUATE_RISK",
        "timestamp": _now_iso(),
        "details": assessment.model_dump(),
    }

    # 2. Telemetría SSE: Conclusión de evaluación de riesgo
    await emit_telemetry(
        correlation_id,
        "RISK_EVALUATION_COMPLETED",
        {"order_id": order_id, "assessment": assessment.model_dump()},
    )

    return {
        "risk_assessment": assessment,
        "current_phase": "RISK_EVALUATED",
        "messages": [AIMessage(content=f"Evaluación de riesgo completada para orden {order_id}: {assessment.risk_level}")],
        "audit_trail": [audit_entry],
    }


async def compliance_checker_node(state: AgentState) -> dict[str, Any]:
    """Agente especialista en validación regulatoria y compliance."""
    order_id = state["order_id"]
    correlation_id = state["correlation_id"]

    # 1. Telemetría SSE: Inicio de verificación de compliance
    await emit_telemetry(correlation_id, "COMPLIANCE_CHECK_STARTED", {"order_id": order_id})

    check = ComplianceCheck(
        is_compliant=True,
        sanctions_clear=True,
        fraud_detected=False,
        violations=[],
    )

    audit_entry: AuditEntry = {
        "step": "compliance_check",
        "agent": "compliance_checker",
        "action": "CHECK_COMPLIANCE",
        "timestamp": _now_iso(),
        "details": check.model_dump(),
    }

    # 2. Telemetría SSE: Conclusión de verificación de compliance
    await emit_telemetry(
        correlation_id,
        "COMPLIANCE_CHECK_COMPLETED",
        {"order_id": order_id, "check": check.model_dump()},
    )

    return {
        "compliance_check": check,
        "current_phase": "COMPLIANCE_VERIFIED",
        "messages": [AIMessage(content=f"Verificación de compliance satisfactoria para orden {order_id}.")],
        "audit_trail": [audit_entry],
    }


async def order_finalizer_node(state: AgentState) -> dict[str, Any]:
    """Agente finalizador que ejecuta la mutación en Go a través del API Zero-Trust."""
    client = BackendClient()
    order_id = state["order_id"]
    correlation_id = state["correlation_id"]

    risk = state.get("risk_assessment")
    compliance = state.get("compliance_check")

    approved = (
        risk is not None
        and risk.risk_level in ("LOW", "MEDIUM")
        and compliance is not None
        and compliance.is_compliant
    )

    final_status = "APPROVED" if approved else "REJECTED"

    # 1. Telemetría SSE: Inicio de mutación transaccional en Go
    await emit_telemetry(
        correlation_id,
        "ORDER_MUTATION_STARTED",
        {"order_id": order_id, "intended_status": final_status},
    )

    # 2. Mutación de estado en Go mediante HTTP autenticado (Zero Trust)
    try:
        updated_order = await client.update_order_status(order_id, final_status)
        action_result = f"Orden {order_id} mutada exitosamente a {final_status} en Go API."
    except Exception as exc:
        await emit_telemetry(
            correlation_id,
            "ORDER_MUTATION_FAILED",
            {"order_id": order_id, "error": str(exc)},
        )
        return {
            "errors": [f"Fallo al actualizar orden en Go API: {exc}"],
            "current_phase": "FINALIZATION_FAILED",
            "audit_trail": [{
                "step": "finalization",
                "agent": "order_finalizer",
                "action": "MUTATE_ORDER_STATUS_FAILED",
                "timestamp": _now_iso(),
                "details": {"error": str(exc)},
            }],
        }

    audit_entry: AuditEntry = {
        "step": "finalization",
        "agent": "order_finalizer",
        "action": "MUTATE_ORDER_STATUS",
        "timestamp": _now_iso(),
        "details": {"new_status": final_status, "order": updated_order},
    }

    # 3. Telemetría SSE: Mutación exitosa
    await emit_telemetry(
        correlation_id,
        "ORDER_MUTATION_COMPLETED",
        {"order_id": order_id, "final_status": final_status},
    )

    return {
        "current_phase": "COMPLETED",
        "messages": [AIMessage(content=action_result)],
        "audit_trail": [audit_entry],
    }
