# Agent Engine: Distributed Multi-Agent Engine (LangGraph)

Asynchronous multi-agent system organized in a star topology (*Supervisor Pattern*) built with LangGraph, LangChain, and Python 3.11+.

## Architecture Highlights
- **Asynchronous Consumption:** Subscribes to the Outbox events topic on Redis with `event_id` deduplication (Idempotence).
- **Star Topology Orchestration:** Central `Supervisor` node enforcing Pydantic `Structured Output` routing with deterministic fallback.
- **Append-Only Immutability:** Global `AgentState` utilizes non-destructive reducer operators (`operator.add`, `add_messages`).
- **Zero Trust Isolation:** Completely isolated from PostgreSQL; entity mutations are strictly performed via `BackendClient` targeting the Go API with Bearer token authentication.
- **Real-Time Telemetry:** Streams granular `astream_events` to Redis `telemetry:{correlation_id}` channels for frontend Server-Sent Events (SSE).
- **Defense-in-Depth:** Semantic and heuristic prompt injection sanitization, complemented by *LLM-as-a-Judge* probabilistic evaluations.

## Local Execution
```bash
pip install -e .
python -m src.consumer.redis_consumer
```

## Testing
```bash
pytest -v tests/
```
