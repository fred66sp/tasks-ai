# Atajos del proyecto. El binario se compila en bin/.

BINARY := tasks-ai
GO     := go

.PHONY: help build test test-race lint lint-fix fmt clean

help: ## Muestra los objetivos disponibles
	@echo "Uso: make [objetivo]"
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*##/ → /'

build: ## Compila el binario en bin/
	$(GO) build -o bin/$(BINARY) ./cmd/$(BINARY)

test: ## Ejecuta las pruebas del proyecto
	$(GO) test ./...

test-race: ## Ejecuta las pruebas buscando carreras
	$(GO) test -race ./...

lint: ## Revisa el código con el linter
	golangci-lint run ./...

lint-fix: ## Revisa el código corrigiendo lo que el linter sepa corregir
	golangci-lint run --fix ./...

fmt: ## Formatea el código
	golangci-lint fmt ./...

clean: ## Retira lo compilado
	rm -rf bin/
