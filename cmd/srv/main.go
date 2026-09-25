package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/Chroq/HashBreaker/pkg"
	"github.com/Chroq/HashBreaker/pkg/db"
	"github.com/Chroq/HashBreaker/pkg/hex"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	depth := flag.Int("depth", 3, "Referential depth precalculation")
	dbDepth := flag.Int("db-depth", 3, "Database batch insert depth")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. PostgreSQL Connection & Migration
	var dbPool *pgxpool.Pool
	dbCfg := db.LoadConfig()
	log.Printf("Connexion à PostgreSQL (%s:%s/%s)...", dbCfg.Host, dbCfg.Port, dbCfg.DBName)
	pool, err := db.Connect(ctx, dbCfg)
	if err != nil {
		log.Printf("⚠️  PostgreSQL indisponible (%v). Le serveur continuera en mode RAM pure.", err)
	} else {
		dbPool = pool
		defer dbPool.Close()

		log.Printf("Application des migrations PostgreSQL...")
		if err := db.Migrate(ctx, dbPool); err != nil {
			log.Fatalf("❌ Erreur de migration : %v", err)
		}
		log.Printf("✅ Migration réussie (Table 'hashes' avec index B-Tree PRIMARY KEY).")

		if *dbDepth > 0 {
			if _, err := db.BatchInsertDepth(ctx, dbPool, *dbDepth); err != nil {
				log.Printf("⚠️  Erreur lors du batch insert DB : %v", err)
			}
		}
	}

	// 2. In-Memory Precalculation
	if *depth > 4 {
		log.Printf("⚠️  Profondeur %d demandée : pré-calcul RAM borné à 4 pour éviter un OOM (>60 Go).", *depth)
	}

	log.Printf("Pré-calcul des référentiels RAM pour profondeur %d...", *depth)
	mapRef := pkg.NewMapReferential(*depth)
	log.Printf("Référentiels prêts. Démarrage du serveur sur :%d", *port)

	// Route standard (In-Memory V5)
	http.HandleFunc("/guess", func(w http.ResponseWriter, r *http.Request) {
		h, err := hex.ParseTargetHash(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		raw := mapRef.GetRaw(h)
		n := 0
		for n < len(raw) && raw[n] != 0 {
			n++
		}
		if n == 0 {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(raw[:n])
	})

	// Route DB (PostgreSQL B-Tree Indexed Lookup)
	http.HandleFunc("/guess/db", func(w http.ResponseWriter, r *http.Request) {
		if dbPool == nil {
			http.Error(w, "Database not configured or unreachable", http.StatusServiceUnavailable)
			return
		}
		h, err := hex.ParseTargetHash(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		word, err := db.GetWordByHash(r.Context(), dbPool, h)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(word))
	})

	addr := fmt.Sprintf(":%d", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Erreur serveur : %v", err)
	}
}
