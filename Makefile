# Makefile Windows PowerShell
.PHONY: help up down logs clean
.PHONY: test test-unit test-integration test-e2e test-all test-coverage
.PHONY: test-db-up test-db-down
.PHONY: build run seed seed-reset lint fmt
.PHONY: load-test load-test-simple
.PHONY: ci

DOCKER_COMPOSE = docker-compose
GO = go
GO_TEST = $(GO) test
GO_TEST_FLAGS = -v -race
COVERAGE_FLAGS = -coverprofile=coverage.out -covermode=atomic


help:
	@echo ""
	@echo "  up                   - Запустить сервис через docker-compose"
	@echo "  down                 - Остановить сервисы (make up)"
	@echo "  logs                 - Показать логи сервисов"
	@echo "  clean                - Полная очистка (make up)"
	@echo "  seed                 - Наполнить БД (make up) тестовыми данными"
	@echo "  seed-reset           - Сбросить и наполнить БД (make up)"
	@echo "  fmt                  - Форматировать код"
	@echo "  test-unit            - Запустить unit-тесты"
	@echo "  test-integration     - Запустить интеграционные тесты (только test-db)"
	@echo "  test-e2e             - Запустить E2E-тесты (make up)"
	@echo "  test-coverage        - Запустить тесты с покрытием (только test-db)"
	@echo "  test-db-up           - Запустить тестовую БД"
	@echo "  test-db-down         - Остановить тестовую БД"
	@echo "  db-shell         	  - Подключение к БД (make up)"
	@echo "  db-shell-test        - Подключение к тестовой БД"
	@echo ""


db-shell
up: 
	@echo "Starting services..."
	$(DOCKER_COMPOSE) up --build -d
	@echo "Services started"

down: 
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down
	@echo "Services stopped"

clean:
	@echo "Cleaning up..."
	$(DOCKER_COMPOSE) down -v 2>nul || true
	-del coverage.out 2>nul
	-del coverage.html 2>nul
	-del cpu.prof 2>nul
	-del mem.prof 2>nul
	-rd /s /q tmp 2>nul
	-rd /s /q bin 2>nul
	@echo "Cleanup complete"

logs: 
	$(DOCKER_COMPOSE) logs -f

seed:
	@echo "Seeding database..."
	docker exec -i booking-db psql -U postgres -d booking_service < migrations/seed_data.sql 2>nul || echo "Database not running. Run 'make up' first"
	@echo "Seed data inserted"

seed-reset:
	@echo "Resetting and seeding database..."
	docker exec booking-db psql -U postgres -d booking_service -c "TRUNCATE bookings, slots, schedules, rooms CASCADE" 2>nul || echo "Database not running"
	docker exec -i booking-db psql -U postgres -d booking_service < migrations/seed_data.sql 2>nul || echo "Database not running"
	@echo "Database reset and seeded"

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted"

test-unit:
	@echo "Running unit tests..."
	$(GO_TEST) $(GO_TEST_FLAGS) ./internal/handler/... ./internal/service/... ./internal/middleware/...
	@echo "Unit tests passed"

test-db-up:
	@echo "Starting test database..."
	-docker stop postgres-test 2>nul
	-docker rm postgres-test 2>nul
	docker run -d --name postgres-test -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=booking_service_test -p 5432:5432 postgres:15
	@echo "Test database started"
	@timeout /t 3 /nobreak >nul

test-db-down:
	@echo "Stopping test database..."
	-docker stop postgres-test 2>nul
	-docker rm postgres-test 2>nul
	@echo "Test database stopped"

test-integration: test-db-up
	@echo "Running integration tests..."
	$(GO_TEST) $(GO_TEST_FLAGS) -tags=integration ./internal/repository/postgres_sql/... -timeout 5m
	$(MAKE) test-db-down
	@echo "Integration tests passed"

test-e2e:
	@echo "Running E2E tests..."
	@echo "Make sure services are running: make up"
	$(GO_TEST) $(GO_TEST_FLAGS) ./tests/e2e/... -timeout 5m
	@echo "E2E tests passed"

test-coverage: test-db-up
	@echo "Running tests with coverage..."
	$(GO_TEST) $(GO_TEST_FLAGS) $(COVERAGE_FLAGS) ./... -timeout 10m
	$(GO) tool cover -html=coverage.out -o coverage.html
	$(GO) tool cover -func=coverage.out
	@echo "Coverage report: coverage.html"
	$(MAKE) test-db-down

test-all: test-unit test-e2e
	@echo "All tests passed"

load-test:
	@echo "Running load test..."
	@echo "Make sure services are running: make up"
	@echo "Make sure k6 is installed: https://k6.io/docs/getting-started/installation/"
	k6 run tests/load_test.js

db-shell:
	@echo "Connecting to database..."
	docker exec -it booking-db psql -U postgres -d booking_service

db-shell-test:
	@echo "Connecting to test database..."
	docker exec -it postgres-test psql -U postgres -d booking_service_test

.DEFAULT_GOAL := help