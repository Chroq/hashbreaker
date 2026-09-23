.PHONY: help build run test bench benchstat load hash guess profile-cpu profile-heap load-and-profile clean

PORT ?= 8080
DEPTH ?= 4
WORD ?= Sh3n
HASH ?= bd7d0ea8cf7ade4a446ba4efc46fd99071ec3f423770991ac51f70ec5a894dc7
RATE ?= 1000
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

hash: ## Calcule le SHA-256 d'un mot (ex: make hash WORD=Sh3n)
	@go run ./cmd/hasher $(WORD)

guess: ## Interroge le serveur HTTP avec une empreinte SHA-256 (ex: make guess HASH=...)
	@curl -s "http://localhost:$(PORT)/guess?hash=$(HASH)"
	@echo ""

test: ## Exécute les tests unitaires
	go test -v ./...

bench: ## Exécute les benchmarks mémoire et CPU
	go test -bench=. -benchmem -count=6 ./pkg

benchstat: ## Exécute les benchmarks et génère l'analyse statistique
	@which benchstat > /dev/null 2>&1 || (echo "Installation de benchstat..." && go install golang.org/x/perf/cmd/benchstat@latest)
	mv bench.txt bench_old.txt
	go test -bench=. -benchmem -count=6 ./pkg > bench.txt
	benchstat bench_old.txt bench.txt
	rm bench_old.txt

load: ## Exécute un test de charge avec Vegeta (ex: make load HASH=... RATE=2000 DURATION=10s)
	@echo "GET http://localhost:$(PORT)/guess?hash=$(HASH)" | vegeta attack -duration=$(DURATION) -rate=$(RATE) | vegeta report

profile-cpu: ## Capture le profil CPU (10s) et ouvre l'interface Web pprof sur :6060
	go tool pprof -http=:$(PPROF_PORT) http://localhost:$(PORT)/debug/pprof/profile?seconds=10

profile-heap: ## Capture le profil Heap (RAM) et ouvre l'interface Web pprof sur :6060
	go tool pprof -http=:$(PPROF_PORT) http://localhost:$(PORT)/debug/pprof/heap

load-and-profile: ## Injecte la charge avec Vegeta et ouvre l'interface Web pprof CPU en parallèle
	@echo "==> Lancement de la charge Vegeta ($(RATE) req/s pendant $(DURATION))..."
	@echo "GET http://localhost:$(PORT)/guess?hash=$(HASH)" | vegeta attack -duration=$(DURATION) -rate=$(RATE) > /tmp/vegeta_results.bin & \
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
