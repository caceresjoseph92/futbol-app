.PHONY: run build test tidy migrate-up

## run: ejecuta el servidor en modo desarrollo
run:
	go run ./cmd/server/main.go

## build: compila el binario
build:
	go build -o bin/futbol-app ./cmd/server/main.go

## test: ejecuta todos los tests unitarios
test:
	go test ./internal/... -v

## test-coverage: tests con cobertura
test-coverage:
	go test ./internal/... -coverprofile=coverage.out
	go tool cover -html=coverage.out

## tidy: limpia y actualiza dependencias
tidy:
	go mod tidy

## migrate-up: aplica las migraciones en orden
migrate-up:
	@echo "Aplicando migraciones..."
	@for f in migrations/*.sql; do \
		echo "→ $$f"; \
		psql $$DATABASE_URL -f $$f; \
	done
	@echo "Migraciones aplicadas."

## lint: revisa el código con go vet
lint:
	go vet ./...

## help: muestra este menú
help:
	@grep -E '^## ' Makefile | sed 's/## //'
