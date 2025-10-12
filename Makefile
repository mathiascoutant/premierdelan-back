.PHONY: run build clean install dev test

# Variables
BINARY_NAME=premier-an-backend
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

# Commandes utiles
fmt:
	@echo "✨ Formatage du code..."
	$(GO) fmt ./...

vet:
	@echo "🔍 Vérification du code..."
	$(GO) vet ./...

lint:
	@echo "🔍 Analyse du code avec golangci-lint..."
	golangci-lint run

deps:
	@echo "📊 Affichage des dépendances..."
	$(GO) list -m all

tidy:
	@echo "🧹 Nettoyage des dépendances..."
	$(GO) mod tidy

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
	@echo "  make run        - Démarrer le serveur"
	@echo "  make build      - Compiler le projet"
	@echo "  make install    - Installer les dépendances"
	@echo "  make dev        - Mode développement (avec air)"
	@echo "  make clean      - Nettoyer les fichiers compilés"
	@echo "  make test       - Exécuter les tests"
	@echo "  make fmt        - Formater le code"
	@echo "  make vet        - Vérifier le code"
	@echo "  make lint       - Analyser le code"
	@echo "  make tidy       - Nettoyer les dépendances"
	@echo "  make db-create  - Créer la base de données"
	@echo "  make db-drop    - Supprimer la base de données"
	@echo "  make db-reset   - Réinitialiser la base de données"

