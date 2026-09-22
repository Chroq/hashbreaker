package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/Chroq/HashBreaker/pkg"
)

type Response struct {
	Word  string `json:"word,omitempty"`
	Found bool   `json:"found"`
}

func parseTargetHash(r *http.Request) ([32]byte, error) {
	word := r.URL.Query().Get("word")
	if word != "" {
		return pkg.GetHash(word), nil
	}

	hashHex := r.URL.Query().Get("hash")
	if hashHex != "" {
		var h [32]byte
		b, err := hex.DecodeString(hashHex)
		if err != nil || len(b) != 32 {
			return h, fmt.Errorf("invalid hash hex")
		}
		copy(h[:], b)
		return h, nil
	}

	return [32]byte{}, fmt.Errorf("missing 'word' or 'hash' query parameter")
}

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	depth := flag.Int("depth", 3, "Referential depth precalculation")
	flag.Parse()

	log.Printf("Pré-calcul des référentiels pour profondeur %d...", *depth)
	mapRef := pkg.NewMapReferential(*depth)
	log.Printf("Référentiels prêts. Démarrage du serveur sur :%d", *port)

	http.HandleFunc("/guess", func(w http.ResponseWriter, r *http.Request) {
		h, err := parseTargetHash(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		word, ok := mapRef.Get(h)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Word:  word,
			Found: ok,
		})
	})

	addr := fmt.Sprintf(":%d", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Erreur serveur : %v", err)
	}
}
