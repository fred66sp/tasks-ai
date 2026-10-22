# Atajos del proyecto. El binario se compila en bin/.

BINARY := tasks-ai
GO     := go

.PHONY: help build install test test-race coverage lint lint-fix fmt check-web clean

help: ## Muestra los objetivos disponibles
	@echo "Uso: make [objetivo]"
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*##/ → /'

build: ## Compila el binario en bin/
	$(GO) build -o bin/$(BINARY) ./cmd/$(BINARY)

install: ## Compila y refresca el binario y la extensión del tablero de este repositorio
	$(GO) build -o bin/$(BINARY) ./cmd/$(BINARY)
	mkdir -p .tasks-ai/bin .tasks-ai/extensions/pi
	cp bin/$(BINARY) .tasks-ai/bin/$(BINARY)
	cp extensions/pi/index.ts extensions/pi/cli.ts extensions/pi/board.ts extensions/pi/package.json .tasks-ai/extensions/pi/
	@test -d extensions/pi/node_modules || { echo "falta extensions/pi/node_modules: ejecuta 'npm install' en extensions/pi"; exit 1; }
	ln -sfn ../../../extensions/pi/node_modules .tasks-ai/extensions/pi/node_modules

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
