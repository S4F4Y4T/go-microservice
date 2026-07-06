A monorepo-based microservices architecture where each service is internally a Modular Monolith (package-by-feature) with independent Go modules.

# Documentation Index

Design decision docs for this Go microservice project. Each doc covers the **what**, **why**, and **alternatives considered**.

| Topic | File | Summary |
|---|---|---|
| **Product Requirements** | [prd.md](prd.md) | What we're building (multi-tenant video upload & streaming SaaS) and why — assumptions, open questions, scope |
| Project Setup & Start | [project-setup.md](project-setup.md) | Makefile, Air hot-reload, environment config, startup flow |
| Folder Structure | [folder-structure.md](folder-structure.md) | Directory layout and what lives where |
| Monorepo & Go Workspace | [monorepo-goworkspace.md](monorepo-goworkspace.md) | Multi-module Go workspace, why monorepo |
| Microservice Architecture | [microservice.md](microservice.md) | Service boundaries, database-per-service, communication |
| Database Migrations | [db-migrate.md](db-migrate.md) | golang-migrate, migration scripts, workflow |
| Dependency Injection | [dependency-injection.md](dependency-injection.md) | Manual DI via bootstrap, why no DI framework |
| Error Handling | [error-handling.md](error-handling.md) | AppError type, error codes, normalization layer |
| Validation | [validation.md](validation.md) | go-playground/validator, field-level errors, JSON tag names |
| Request Decoding | [request.md](request.md) | JSON decoding, size limits, unknown field rejection |
| Response Format | [response.md](response.md) | Envelope structure, success/error shapes |
| Pagination | [pagination.md](pagination.md) | Page+limit params, clamping, meta response |
| Filter & Sort | [filter-sort.md](filter-sort.md) | Allowlist schema, query params, GORM scopes |
| Rate Limiting | [rate-limiting.md](rate-limiting.md) | Redis fixed-window, per-IP, fail-open |
| Logger | [logger.md](logger.md) | slog-based structured JSON, request-scoped, context propagation |
| Middleware | [middleware.md](middleware.md) | Chain order, what each middleware does and why |
| Router | [router.md](router.md) | net/http ServeMux, versioned routes, health endpoints |
| DTO | [dto.md](dto.md) | Request/response DTOs, separation from domain model |
| Model & Repository Interface | [model.md](model.md) | Domain models, repository interface on the model layer |
| Repository | [repository.md](repository.md) | GORM implementation, unique violation handling |
| Service Layer | [service.md](service.md) | Business logic, pre-checks, orchestration |
| Handler | [handler.md](handler.md) | HTTP handlers, decoding → validation → service → response |
| Authentication | [auth.md](auth.md) | JWT + refresh token, Redis store, token rotation, auth middleware |
| HTTP Server & Graceful Shutdown | [http-server.md](http-server.md) | Timeouts (read/write/idle), MaxHeaderBytes, SIGTERM drain |
| Docker & Containerization | [docker.md](docker.md) | Dev images, docker-compose, port strategy, health checks |
| API Gateway | [api-gateway.md](api-gateway.md) | Kong DB-less mode, routing, CORS/rate-limit/correlation-ID plugins |
| gRPC (East-West) | [grpc.md](grpc.md) | auth ↔ user service contract, server/client wiring, error-code mapping, known gaps |
| API Docs (Swagger UI) | [api-docs.md](api-docs.md) | Dedicated docs service, single combined spec, served through Kong |
| Internal Architecture & Alternatives | [internal-architecture.md](internal-architecture.md) | Modular Monolith (chosen) vs Layered vs Clean vs Hexagonal vs DDD vs CQRS — when to choose which |
| User Service — Modular Monolith | [user-service-architecture.md](user-service-architecture.md) | Module map (user/auth/health), the auth→user coupling to watch, DDD-lite rationale, improvement suggestions |
| Testing | [testing.md](testing.md) | Table-driven `pkg/` tests, `auth.Service`/`Handler` tests via hand-written fakes, what's not covered yet |
| **Roadmap** | [next.md](next.md) | Phase-by-phase checklist for large-scale production (auth, gRPC, Kafka, k8s, observability…) |
