.PHONY: run build clean install dev test deps-check deps-update deps-vuln quality

# Variables
BINARY_NAME=backend
GO=go
GOFLAGS=-v

# Commandes principales
run:
	@echo "🚀 Démarrage du serveur..."
	$(GO) run main.go

build:
	@echo "🔨 Compilation du projet..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .
	@echo "✓ Compilation terminée: ./$(BINARY_NAME)"

install:
	@echo "📦 Installation des dépendances..."
	$(GO) mod download
	@echo "✓ Dépendances installées"

dev:
	@echo "🔧 Mode développement avec rechargement automatique..."
	@echo "Installez air si ce n'est pas fait: go install github.com/cosmtrek/air@latest"
	air

clean:
	@echo "🧹 Nettoyage des fichiers compilés..."
	@rm -f $(BINARY_NAME)
	@rm -rf tmp/
	@echo "✓ Nettoyage terminé"

test:
	@echo "🧪 Exécution des tests..."
	$(GO) test -v ./...

# Qualité : exécute vet, lint et test
quality:
	@echo "🔍 Contrôle qualité du code..."
	@$(GO) vet ./...
	@echo "✓ go vet OK"
	@GOLANGCI=$$(command -v golangci-lint 2>/dev/null || echo "$(shell go env GOPATH)/bin/golangci-lint"); \
	if [ -x "$$GOLANGCI" ]; then \
		$$GOLANGCI run && echo "✓ golangci-lint OK"; \
	else \
		echo "⚠️  golangci-lint non installé (optionnel): go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi
	@$(GO) test ./... -count=1
	@echo "✓ Tests OK"
	@echo "✅ Contrôle qualité terminé"

# Gestion des dépendances
deps-check:
	@echo "🔍 Vérification des dépendances..."
	@chmod +x scripts/check-dependencies.sh
	@./scripts/check-dependencies.sh

deps-update:
	@echo "🔄 Mise à jour des dépendances..."
	@echo "⚠️  Attention: cette commande mettra à jour toutes les dépendances"
	@read -p "Voulez-vous continuer? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "✅ Dépendances mises à jour"
	@echo "📝 N'oubliez pas de tester et de commiter les changements"

deps-update-minor:
	@echo "🔄 Mise à jour des dépendances (mineures et patches uniquement)..."
	@echo "⚠️  Certaines dépendances peuvent échouer (versions non disponibles)"
	@$(GO) get -u=patch ./... || echo "⚠️  Certaines mises à jour ont échoué (normal si versions non disponibles)"
	@$(GO) mod tidy
	@echo "✅ Processus de mise à jour terminé"
	@echo "💡 Vérifiez les changements avec: git diff go.mod go.sum"

deps-vuln:
	@echo "🔒 Vérification des vulnérabilités..."
	@GOVULNCHECK_CMD=$$(command -v govulncheck 2>/dev/null || echo "$(shell go env GOPATH)/bin/govulncheck"); \
	if [ -f "$$GOVULNCHECK_CMD" ] || command -v govulncheck &> /dev/null; then \
		$$GOVULNCHECK_CMD ./... 2>&1 || echo "⚠️  Des vulnérabilités ont été détectées"; \
	else \
		echo "⚠️  govulncheck n'est pas installé"; \
		echo ""; \
		echo "💡 Pour installer:"; \
		echo "   ./scripts/install-govulncheck.sh"; \
		echo "   ou: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		echo ""; \
		echo "   Puis ajouter au PATH: export PATH=\$$PATH:$$(go env GOPATH)/bin"; \
	fi

# Commandes utiles
fmt:
	@echo "✨ Formatage du code..."
	$(GO) fmt ./...

vet:
	@echo "🔍 Vérification du code..."
	$(GO) vet ./...

lint:
	@echo "🔍 Analyse du code avec golangci-lint..."
	@GOLANGCI=$$(command -v golangci-lint 2>/dev/null || echo "$(shell go env GOPATH)/bin/golangci-lint"); \
	if [ -x "$$GOLANGCI" ]; then \
		$$GOLANGCI run; \
	else \
		echo "⚠️  golangci-lint non installé"; \
		echo "   Installation: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "   Puis: export PATH=\$$PATH:$$(go env GOPATH)/bin"; \
		exit 1; \
	fi

# Base de données
db-create:
	@echo "🗄️  Création de la base de données..."
	@createdb premier_an_db || echo "Base de données déjà existante"

db-drop:
	@echo "🗑️  Suppression de la base de données..."
	@dropdb premier_an_db || echo "Base de données n'existe pas"

db-reset: db-drop db-create
	@echo "✓ Base de données réinitialisée"

# Aide
help:
	@echo "Commandes disponibles:"
	@echo "  make run              - Démarrer le serveur"
	@echo "  make build            - Compiler le projet"
	@echo "  make install          - Installer les dépendances"
	@echo "  make dev              - Mode développement (avec air)"
	@echo "  make clean            - Nettoyer les fichiers compilés"
	@echo "  make test             - Exécuter les tests"
	@echo "  make deps-check       - Vérifier l'état des dépendances"
	@echo "  make deps-update      - Mettre à jour toutes les dépendances"
	@echo "  make deps-update-minor - Mettre à jour (patches/mineures)"
	@echo "  make deps-vuln        - Vérifier les vulnérabilités"
	@echo "  make fmt              - Formater le code"
	@echo "  make vet              - Vérifier le code"
	@echo "  make lint             - Analyser le code"
	@echo "  make quality          - Vet + Lint + Tests"
