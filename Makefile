.PHONY: help build run test bench benchstat load profile-cpu profile-heap load-and-profile clean

PORT ?= 8080
DEPTH ?= 3
WORD ?= z3D
RATE ?= 2000
DURATION ?= 15s
PPROF_PORT ?= 6060
BIN_DIR := bin
BINARY := $(BIN_DIR)/hashbreaker-server

help: ## Affiche l'aide
	@echo "Commandes disponibles :"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

build: ## Compile le serveur HTTP
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) ./cmd/srv

run: build ## Compile et démarre le serveur HTTP (PORT=8080 DEPTH=3)
	./$(BINARY) -port $(PORT) -depth $(DEPTH)

test: ## Exécute les tests unitaires
	go test -v ./...

bench: ## Exécute les benchmarks mémoire et CPU
	go test -bench=. -benchmem -count=6 ./pkg

benchstat: ## Exécute les benchmarks et génère l'analyse statistique
	@which benchstat > /dev/null 2>&1 || (echo "Installation de benchstat..." && go install golang.org/x/perf/cmd/benchstat@latest)
	go test -bench=. -benchmem -count=6 ./pkg > bench.txt
	benchstat bench.txt

load: ## Exécute un test de charge avec Vegeta (ex: make load WORD=z3D RATE=2000 DURATION=10s)
	@echo "GET http://localhost:$(PORT)/guess?word=$(WORD)" | vegeta attack -duration=$(DURATION) -rate=$(RATE) | vegeta report

profile-cpu: ## Capture le profil CPU (10s) et ouvre l'interface Web pprof sur :6060
	go tool pprof -http=:$(PPROF_PORT) http://localhost:$(PORT)/debug/pprof/profile?seconds=10

profile-heap: ## Capture le profil Heap (RAM) et ouvre l'interface Web pprof sur :6060
	go tool pprof -http=:$(PPROF_PORT) http://localhost:$(PORT)/debug/pprof/heap

load-and-profile: ## Injecte la charge avec Vegeta et ouvre l'interface Web pprof CPU en parallèle
	@echo "==> Lancement de la charge Vegeta ($(RATE) req/s pendant $(DURATION))..."
	@echo "GET http://localhost:$(PORT)/guess?word=$(WORD)" | vegeta attack -duration=$(DURATION) -rate=$(RATE) > /tmp/vegeta_results.bin & \
	ATTACK_PID=$$!; \
	sleep 1; \
	echo "==> Capture du profil CPU (10s) et ouverture de l'interface pprof sur :$(PPROF_PORT)..."; \
	go tool pprof -http=:$(PPROF_PORT) http://localhost:$(PORT)/debug/pprof/profile?seconds=10; \
	wait $$ATTACK_PID; \
	echo "==> Rapport Vegeta :"; \
	vegeta report < /tmp/vegeta_results.bin; \
	rm -f /tmp/vegeta_results.bin

clean: ## Nettoie les binaires et fichiers temporaires
	rm -rf $(BIN_DIR) bench.txt /tmp/vegeta_results.bin
