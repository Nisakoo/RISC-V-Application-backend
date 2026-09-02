# Backend

Go REST API for the RISC-V instruction catalogue and program execution requests.

## Structure

- `cmd/api` — application entry point;
- `internal/config` — environment configuration;
- `internal/domain` — RISC-V domain models and DTOs;
- `internal/handler` — HTTP handlers;
- `internal/service` — use cases and execution logic;
- `internal/repository` — PostgreSQL and MinIO access;
- `internal/app` — dependency wiring and server lifecycle;
- `migrations` — database migrations;
- `docker` — local infrastructure configuration.

## Local infrastructure

MinIO is defined in [`docker/compose.yaml`](docker/compose.yaml). Create `docker/.env` from `docker/.env.example`, set non-default credentials, then run:

```sh
cd docker
docker compose up -d
```

MinIO API: `http://localhost:9000`; console: `http://localhost:9001`.
