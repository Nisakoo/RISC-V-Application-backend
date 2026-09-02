# Backend

Go REST API for the RISC-V instruction catalogue and program execution requests.

## Lab 1

The first lab is a server-rendered interface without JavaScript. It uses one in-memory collection of RISC-V instructions and three GET controllers:

- `/feed` and `/feed/:id` — instruction feed; `?next=true` opens the next visible instruction;
- `/instruction/draft` — the single draft instruction;
- `/instructions?operands=3` — instruction grid with server-side filtering by operand count.

Run it from this directory:

```sh
go run ./cmd/api
```

The server listens on `http://localhost:8080`. Images and videos are read from the public local MinIO bucket `riscv-assets`.

## Structure

- `cmd/api` — application entry point;
- `internal/domain` — RISC-V domain models and DTOs;
- `internal/handler` — HTTP handlers;
- `internal/repository` — PostgreSQL and MinIO access;
- `internal/app` — dependency wiring and server lifecycle;
- `docker` — local infrastructure configuration.

## Local infrastructure

MinIO is defined in [`docker/compose.yaml`](docker/compose.yaml). Create `docker/.env` from `docker/.env.example`, set non-default credentials, then run from the workspace root:

```sh
make infra-up
```

The setup container creates the public `riscv-assets` bucket and copies all images and videos from the root `assets` directory into it.

MinIO API: `http://localhost:9000`; console: `http://localhost:9001`.
