<p align="center">
  <img width="256" height="256" alt="二次元卡通融合logo设计 (10)" src="https://github.com/user-attachments/assets/7dd6dd0d-10cc-431e-b164-06f307a2b44a" />
</p>

# OpenOS

**An operating environment for AI agents** — API-first, cloud-native, and built for isolation, scheduling, and observability. System kernel and foundational framework implementation completed.

This repository contains the **Agent OS** design documents and implementation workstream. For deeper technical detail, see [`agent-os/architecture/`](agent-os/architecture/) and [`agent-os/technical-design/`](agent-os/technical-design/).

---

## Design philosophy

Agent OS is guided by principles that distinguish an *agent-native* platform from a traditional user-centric stack:

1. **API-first** — Capabilities are exposed through APIs (REST, gRPC, WebSocket); there is no dependency on a GUI for core operations.
2. **Agent-centric** — Scheduling, lifecycle, and resources are modeled around agents, not around interactive sessions.
3. **Security by design** — Multi-tenant boundaries, least privilege, and strong isolation (containers/sandboxes) are first-class concerns.
4. **Cloud-native** — Microservice-friendly boundaries, containerized workloads, and elastic scaling.
5. **Observability** — Metrics, traces, and structured events are required for operating agents at scale.

**Product stance** (from the product definition): agents run **continuously**, need **strict isolation**, and collaborate through **APIs and messaging** rather than GUIs. The system favors **declarative configuration**, **event-driven** integration, and **safe defaults**.

**Governance stance** (from architecture docs): capabilities are tracked under a **single source of truth** using three lenses — **Target** (long-term), **IterationScope** (committed for the current iteration), and **Implemented** (verified in the repo). Items only move to **Implemented** when **acceptance criteria** and **evidence links** are satisfied — not when documentation alone exists.

**Data philosophy**: a **thin data layer** (`internal/database` + `internal/data/*`) keeps repositories, transactions (Unit of Work), outbox, schema compatibility, and migrations explicit **without** introducing a heavyweight standalone “data microservice” in early phases.

---

## Technical architecture (summary)

The high-level stack is organized in **layers** (see [`agent-os/architecture/system-architecture.md`](agent-os/architecture/system-architecture.md) and [`agent-os/architecture/tech-architecture.md`](agent-os/architecture/tech-architecture.md)):

| Layer | Role (illustrative) |
|--------|---------------------|
| **Interface** | API gateway, CLI, dashboard — REST/OpenAPI, gRPC, WebSocket |
| **Orchestration** | Workflow engine, service discovery, event bus |
| **Agent** | Agent runtime, lifecycle manager, registry |
| **Platform** | Resource scheduler, security enforcement, networking |
| **Core services** | Storage, messaging, monitoring |

**Interface contracts (v1 baseline)** include a unified error model, **idempotency** (`Idempotency-Key` on create-style APIs), path versioning (e.g. `/api/v1/...`), explicit **RBAC + tenant** requirements per API, and **event payload** minimum fields (`event_id`, `event_type`, `schema_version`, `occurred_at`, `trace_id`, `agent_id`, `tenant_id`). Cross-protocol rules are summarized in [`agent-os/architecture/api-compatibility-matrix.md`](agent-os/architecture/api-compatibility-matrix.md).

**Orchestration** is specified with a **minimal control-plane state machine** (Created → Scheduled → Starting → Ready / Failed, with recovery and compensation) and a **consistency / outbox state machine** (Persisted → OutboxWritten → Published → Acknowledged, with replay and dead-letter paths). The two must align: e.g. control flow should not reach **Ready** unless the consistency stream has at least reached a **Published**-class state where required.

**Thin data layer** (see [`agent-os/architecture/data-layer-blueprint.md`](agent-os/architecture/data-layer-blueprint.md)): repositories hide SQL from upper layers; **Unit of Work** bundles business writes, outbox writes, and audit writes; **OutboxPublisher** handles delivery semantics only; **SchemaRegistry** owns `schema_version` and compatibility checks.

**MVP / current implementation note**: the long-term interface story includes Envoy and a rich gateway; the **near-term** path documented in-repo uses an **in-process HTTP server** in Go (`internal/server`) for health, metrics, and baseline routing, with room to evolve toward a sidecar or standalone gateway.

**Messaging ADR**: NATS-first messaging is recorded under [`agent-os/architecture/adr/`](agent-os/architecture/adr/).

**Stack highlights** (from architecture docs): **Go** for the core, **Kubernetes-oriented** scheduling and platform concepts, **PostgreSQL / Redis / S3-compatible** storage patterns, **NATS/Kafka-compatible** messaging, **Prometheus-compatible** metrics.

---

## Validation, SLO targets & release posture

This project does **not** define a financial “backtesting” workflow. Instead, **reliability and quality are framed as measurable SLIs/SLOs, CI data gates, and release gates** so that behavior can be validated before and after rollout.

### Key SLI / SLO targets (documented baselines)

| Area | SLI | Target | Window |
|------|-----|--------|--------|
| Agent start | Success rate | ≥ 99.0% | 7-day rolling |
| Agent start | P95 latency | ≤ 5s | 24h |
| Control-plane API | Error rate | ≤ 1.0% | 24h |
| Outbox delivery | Success rate | ≥ 99.5% | 24h |
| Event consumption | ACK latency P95 | ≤ 2s | 24h |

**Error budget** rules tie SLO breaches to release policy (freeze features, gate releases, or allow only fixes/rollbacks). Details: [`agent-os/architecture/slo-release-gate.md`](agent-os/architecture/slo-release-gate.md).

### CI data gates (automation targets)

Gate-1–4 (migrations, schema compat, outbox/idempotency, idempotency key policy) are intended to **block merges** when violated; Gate-5–6 (observability fields, evidence for high-risk items) **block release** until satisfied. See [`agent-os/architecture/ci-data-gates.md`](agent-os/architecture/ci-data-gates.md).

### Product-level success metrics (aspirational)

The product summary cites targets such as **API P95 &lt; 50ms**, **availability ≥ 99.9%**, **resource utilization ≥ 75%**, and **1000+ concurrent agents** — as **north-star** engineering goals, not a guarantee of current measurements. See [`agent-os/summary-overview.md`](agent-os/summary-overview.md).

---

## Project status & what is delivered

### Versioning

The implementation build is aligned with **v0.1.0** (see `VERSION` in [`agent-os/implementation/Makefile`](agent-os/implementation/Makefile)).

### What v0.1.0 represents

- **Kernel-oriented foundation**: runtime interfaces, substantial **containerd** integration work, partial **gVisor** integration, and shared **lifecycle / types / security** modeling in the runtime area.
- **Framework & operations baseline**: **configuration** system with tests, **health** checks with tests, **HTTP server** skeleton (`internal/server`) with health/metrics-style endpoints, **scheduler interfaces** and scaffolding (algorithms and full policy still evolving), **basic monitoring** hooks.
- **Architecture & governance artifacts**: layered architecture, dual state machines, thin data layer blueprint, API compatibility matrix, ADRs (e.g. API gateway MVP path, NATS-first messaging), SLO/release and CI data gate specifications, and capability tracking (Target / IterationScope / Implemented).

### MVP scope (planning)

The MVP timeline was adjusted to a **6-month** horizon with phased delivery (technical validation → core components → integration and testing). See [`agent-os/mvp-adjustment-summary.md`](agent-os/mvp-adjustment-summary.md).

### Honest gap summary (from implementation reviews)

End-to-end **workflow engine**, **service discovery**, full **gateway business APIs**, **scheduler algorithms**, **policy enforcement**, **persistent data layer**, and **NATS/Kafka** integration are **not** complete as of the documented reviews; the project is in **early implementation** with a strong specification and partial runtime/config/server code. Refer to [`agent-os/current-project-state.md`](agent-os/current-project-state.md) and [`agent-os/implementation-status-analysis.md`](agent-os/implementation-status-analysis.md).

---

## Documentation map

| Topic | Location |
|--------|----------|
| System architecture | [`agent-os/architecture/system-architecture.md`](agent-os/architecture/system-architecture.md) |
| Detailed technical architecture | [`agent-os/architecture/tech-architecture.md`](agent-os/architecture/tech-architecture.md) |
| Thin data layer | [`agent-os/architecture/data-layer-blueprint.md`](agent-os/architecture/data-layer-blueprint.md) |
| Product vision | [`agent-os/product-vision.md`](agent-os/product-vision.md) |
| Summary overview | [`agent-os/summary-overview.md`](agent-os/summary-overview.md) |

---

## License

GPL-3.0
