SHELL := /bin/sh

RISC_V_GO_CACHE ?= /tmp/riscv-go-build-cache
COMPOSE_FILE := docker/compose.yaml
COMPOSE_ENV := docker/.env

.DEFAULT_GOAL := help

.PHONY: help install infra-up infra-down run format test test-backend test-media

help:
	@echo "Доступные команды:"
	@echo "  make install        Установить зависимости Go"
	@echo "  make infra-up       Запустить MinIO и загрузить медиа"
	@echo "  make infra-down     Остановить MinIO"
	@echo "  make run            Запустить первую лабораторную на :8080"
	@echo "  make format         Отформатировать Go-код"
	@echo "  make test-backend   Проверить форматирование, go vet и Go-тесты"
	@echo "  make test-media     Проверить доступность медиа в MinIO"
	@echo "  make test           Выполнить все проверки backend"

install:
	go mod download

infra-up:
	test -f $(COMPOSE_ENV) || (echo "Создайте $(COMPOSE_ENV) на основе docker/.env.example" && exit 1)
	docker compose --env-file $(COMPOSE_ENV) -f $(COMPOSE_FILE) up -d minio
	docker compose --env-file $(COMPOSE_ENV) -f $(COMPOSE_FILE) run --rm minio-setup

infra-down:
	docker compose --env-file $(COMPOSE_ENV) -f $(COMPOSE_FILE) down

run: infra-up
	go run ./cmd/api

format:
	gofmt -w $$(find . -path ./.git -prune -o -type f -name '*.go' -print)

test-backend:
	@test -z "$$(gofmt -l $$(find . -path ./.git -prune -o -type f -name '*.go' -print))" || (echo "Go-файлы не отформатированы. Выполните make format" && exit 1)
	GOCACHE=$(RISC_V_GO_CACHE) go vet ./...
	GOCACHE=$(RISC_V_GO_CACHE) go test ./...

test-media: infra-up
	@curl --retry 5 --retry-delay 1 --retry-all-errors -fsSI "http://localhost:9000/riscv-assets/images/default.svg" | grep -qi '^content-type: image/svg+xml' || (echo "Изображение по умолчанию недоступно в MinIO" && exit 1)
	@for instruction in ADD SUB ADDI BEQ LUI JAL; do \
		curl --retry 5 --retry-delay 1 --retry-all-errors -fsSI "http://localhost:9000/riscv-assets/images/$$instruction.png" | grep -qi '^content-type: image/png' || (echo "Изображение $$instruction недоступно в MinIO" && exit 1); \
		curl --retry 5 --retry-delay 1 --retry-all-errors -fsSI "http://localhost:9000/riscv-assets/video/$$instruction.mp4" | grep -qi '^content-type: video/mp4' || (echo "Видео $$instruction недоступно в MinIO" && exit 1); \
	done

test: test-backend test-media
