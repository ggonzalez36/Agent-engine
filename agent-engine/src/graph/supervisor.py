"""Supervisor de LangGraph con Topología en Estrella y Enrutamiento Estructurado."""

from __future__ import annotations

import json
import os
from typing import Any
import redis.asyncio as aioredis
from langchain_core.messages import SystemMessage
from langchain_openai import ChatOpenAI
from langgraph.graph import END, START, StateGraph
from src.graph.nodes import (
    compliance_checker_node,
    order_finalizer_node,
    risk_evaluator_node,
)
from src.state.agent_state import AgentState, RouterDecision

SUPERVISOR_PROMPT = """Eres el Supervisor Orquestador del sistema multiagente.
Tu responsabilidad es evaluar el estado actual de la orden y enrutar el flujo hacia el especialista adecuado:
1. 'risk_evaluator': Si no existe evaluación de riesgo (risk_assessment es None).
2. 'compliance_checker': Si el riesgo ya fue evaluado pero no se ha validado compliance (compliance_check es None).
3. 'order_finalizer': Si tanto riesgo como compliance han sido evaluados y la orden aún no está finalizada.
4. 'FINISH': Si la orden ya fue procesada y finalizada exitosamente o si se presentaron errores críticos.

Debes responder ÚNICAMENTE con la estructura de salida requerida (RouterDecision)."""

DEFAULT_RECURSION_LIMIT: int = 15


class SupervisorGraph:
    def __init__(self, redis_url: str | None = None) -> None:
        self.redis_url = redis_url or os.getenv("REDIS_URL", "redis://localhost:6379/0")
        self.redis_client = aioredis.from_url(self.redis_url, decode_responses=True)
        openai_key = os.getenv("OPENAI_API_KEY", "")
        if openai_key:
            self.llm = ChatOpenAI(
                model=os.getenv("OPENAI_MODEL_NAME", "gpt-4o-mini"),
                temperature=0.0,
                api_key=openai_key,
            ).with_structured_output(RouterDecision)
        else:
            self.llm = None

    def _deterministic_routing(self, state: AgentState) -> RouterDecision:
        """Enrutamiento heurístico/determinista cuando el LLM no está configurado o falla."""
        if state.get("risk_assessment") is None:
            return RouterDecision(next_node="risk_evaluator", reasoning="Falta evaluación cuantitativa de riesgo", confidence=1.0)
        elif state.get("compliance_check") is None:
            return RouterDecision(next_node="compliance_checker", reasoning="Falta verificación regulatoria de compliance", confidence=1.0)
        elif state.get("current_phase") != "COMPLETED" and not state.get("errors"):
            return RouterDecision(next_node="order_finalizer", reasoning="Riesgo y compliance validados; finalizar orden en Go API", confidence=1.0)
        else:
            return RouterDecision(next_node="FINISH", reasoning="Flujo agéntico completado o detenido por errores", confidence=1.0)

    async def emit_telemetry(self, correlation_id: str, event_type: str, data: dict[str, Any]) -> None:
        """Publica telemetría granular en Redis Pub/Sub consumida por el backend Go vía SSE."""
        payload = json.dumps({
            "event_type": event_type,
            "correlation_id": correlation_id,
            "timestamp": os.getenv("CURRENT_TIME", ""),
            "data": data,
        })
        try:
            await self.redis_client.publish(f"telemetry:{correlation_id}", payload)
        except Exception:
            pass

    async def supervisor_node(self, state: AgentState) -> dict[str, Any]:
        """Nodo orquestador que decide el siguiente paso mediante Structured Output."""
        correlation_id = state["correlation_id"]
        order_id = state["order_id"]

        # Telemetría de inicio de deliberación
        await self.emit_telemetry(
            correlation_id,
            "SUPERVISOR_EVALUATING",
            {"order_id": order_id, "current_phase": state.get("current_phase")},
        )

        # Evaluación determinista / LLM estructurado
        if self.llm is not None:
            messages = [
                SystemMessage(content=SUPERVISOR_PROMPT),
                *state["messages"],
            ]
            try:
                decision: RouterDecision = await self.llm.ainvoke(messages)
            except Exception:
                decision = self._deterministic_routing(state)
        else:
            decision = self._deterministic_routing(state)

        # Telemetría de enrutamiento
        await self.emit_telemetry(
            correlation_id,
            "SUPERVISOR_ROUTED",
            {"decision": decision.model_dump()},
        )
        
        return {
            "current_phase": f"ROUTED_TO_{decision.next_node}",
            "audit_trail": [{
                "step": "supervisor_routing",
                "agent": "supervisor",
                "action": "ROUTE",
                "timestamp": "",
                "details": decision.model_dump(),
            }],
        }

    def build_graph(self):
        """Construye la topología en estrella del StateGraph."""
        builder = StateGraph(AgentState)

        # Registro de nodos
        builder.add_node("supervisor", self.supervisor_node)
        builder.add_node("risk_evaluator", risk_evaluator_node)
        builder.add_node("compliance_checker", compliance_checker_node)
        builder.add_node("order_finalizer", order_finalizer_node)

        # Entrada inicial al Supervisor
        builder.add_edge(START, "supervisor")

        # Función de enrutamiento condicional desde el Supervisor
        def route_next(state: AgentState) -> str:
            last_audit = state["audit_trail"][-1]["details"]
            next_step = last_audit.get("next_node")
            if next_step == "FINISH" or not next_step:
                return END
            return next_step

        builder.add_conditional_edges(
            "supervisor",
            route_next,
            {
                "risk_evaluator": "risk_evaluator",
                "compliance_checker": "compliance_checker",
                "order_finalizer": "order_finalizer",
                END: END,
            },
        )

        # Topología en estrella: todos los nodos retornan al Supervisor para el siguiente ciclo
        builder.add_edge("risk_evaluator", "supervisor")
        builder.add_edge("compliance_checker", "supervisor")
        builder.add_edge("order_finalizer", "supervisor")

        return builder.compile()


async def run_agent_workflow(
    initial_state: AgentState,
    recursion_limit: int = DEFAULT_RECURSION_LIMIT,
) -> AgentState:
    """Ejecuta el grafo multiagente transmitiendo eventos granulares vía astream_events (Pilar 3)."""
    supervisor = SupervisorGraph()
    graph = supervisor.build_graph()
    correlation_id = initial_state.get("correlation_id", "corr_unknown")

    final_state: AgentState = initial_state

    # Transmisión en tiempo real de eventos reactivos del grafo
    try:
        async for event in graph.astream_events(
            initial_state,
            version="v2",
            config={"recursion_limit": recursion_limit},
        ):
            event_type = event.get("event", "")
            # Publicación de telemetría reactiva en tiempo real hacia Redis
            if event_type in ("on_chain_start", "on_chain_end", "on_chat_model_stream", "on_tool_start"):
                node_name = event.get("name", "graph")
                await supervisor.emit_telemetry(
                    correlation_id=correlation_id,
                    event_type=f"LANGGRAPH_{event_type.upper()}",
                    data={"node": node_name, "run_id": str(event.get("run_id", ""))},
                )
            # Captura del estado final en la conclusión del grafo raíz
            if event_type == "on_chain_end" and event.get("name") == "LangGraph":
                output = event.get("data", {}).get("output")
                if isinstance(output, dict):
                    final_state = output  # type: ignore[assignment]
    except Exception:
        # Fallback de seguridad mediante ainvoke si astream_events sufre corte
        final_state = await graph.ainvoke(
            initial_state,
            config={"recursion_limit": recursion_limit},
        )

    return final_state
