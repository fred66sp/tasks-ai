# Atajos del proyecto. El binario se compila en bin/.

BINARY := tasks-ai
GO     := go

.PHONY: help build test test-race coverage lint lint-fix fmt check-web clean

help: ## Muestra los objetivos disponibles
	@echo "Uso: make [objetivo]"
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*##/ → /'

build: ## Compila el binario en bin/
	$(GO) build -o bin/$(BINARY) ./cmd/$(BINARY)

test: ## Ejecuta las pruebas del proyecto
	$(GO) test ./...

test-race: ## Ejecuta las pruebas buscando carreras
	$(GO) test -race ./...

coverage: ## Mide la cobertura de las pruebas y deja el perfil en coverage/
	@mkdir -p coverage
	$(GO) test ./... -coverprofile=coverage/profile.out
	$(GO) tool cover -func=coverage/profile.out

lint: ## Revisa el código con el linter
	golangci-lint run ./...

lint-fix: ## Revisa el código corrigiendo lo que el linter sepa corregir
	golangci-lint run --fix ./...

fmt: ## Formatea el código
	golangci-lint fmt ./...

check-web: ## Comprueba que la interfaz construida coincide con su fuente
	cd web && npm run build
	git diff --exit-code -- web/public

clean: ## Retira lo compilado
	rm -rf bin/
