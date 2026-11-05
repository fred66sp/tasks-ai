# Atajos del proyecto. El binario se compila donde el tablero lo ejecuta.

BINARY := tasks-ai
GO     := go

.PHONY: help install test test-race coverage lint lint-fix fmt check-web docs-vendor clean

help: ## Muestra los objetivos disponibles
	@echo "Uso: make [objetivo]"
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*##/ → /'

install: ## Compila y refresca el binario, las extensiones y el contrato del tablero de este repositorio
	mkdir -p .tasks-ai/bin .tasks-ai/extensions/pi .tasks-ai/extensions/opencode
	$(GO) build -o .tasks-ai/bin/$(BINARY) ./cmd/$(BINARY)
	cp extensions/pi/index.ts extensions/pi/cli.ts extensions/pi/board.ts extensions/pi/package.json .tasks-ai/extensions/pi/
	cp extensions/opencode/index.ts extensions/opencode/tasks.ts extensions/opencode/package.json .tasks-ai/extensions/opencode/
	cp internal/install/contract_tags.md .tasks-ai/CONTRATO_ETIQUETAS.md
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

check-web: ## Comprueba las pruebas de la interfaz y que lo construido coincide con su fuente
	cd web && npm test && npm run build
	git diff --exit-code -- web/public

docs-vendor: ## Refresca las copias vendorizadas de docs/web desde sus dueños (ADR-018)
	cp web/public/icons/icons.css docs/web/assets/vendor/icons/icons.css
	rm -rf docs/web/assets/vendor/icons/heroicons
	cp -r web/public/icons/heroicons docs/web/assets/vendor/icons/heroicons
	cp web/public/task-manager/app-vue.css docs/web/assets/vendor/app-vue.css
	cp web/public/vendor/vue.global.prod.js docs/web/assets/vendor/vue.global.prod.js
	cp web/public/vendor/toast-anchor.mjs docs/web/assets/vendor/toast-anchor/toast-anchor.mjs

clean: ## Retira lo que generan las pruebas
	rm -rf coverage/
