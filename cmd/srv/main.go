package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/Chroq/HashBreaker/pkg"
	"github.com/Chroq/HashBreaker/pkg/hex"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	depth := flag.Int("depth", 3, "Referential depth precalculation")
	flag.Parse()

	if *depth > 4 {
		log.Printf("⚠️  Profondeur %d demandée : pré-calcul RAM borné à 4 pour éviter un OOM (>60 Go).", *depth)
	}

	log.Printf("Pré-calcul des référentiels pour profondeur %d...", *depth)
	mapRef := pkg.NewMapReferential(*depth)
	log.Printf("Référentiels prêts. Démarrage du serveur sur :%d", *port)

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

	addr := fmt.Sprintf(":%d", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Erreur serveur : %v", err)
	}
}
