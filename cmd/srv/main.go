package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/Chroq/HashBreaker/pkg"
)

func fromHexChar(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	default:
		return 255
	}
}

func decodeHex32(src string) ([32]byte, error) {
	var dst [32]byte
	if len(src) != 64 {
		return dst, fmt.Errorf("invalid hash hex length: expected 64 hex characters")
	}
	for i := 0; i < 32; i++ {
		hi := fromHexChar(src[2*i])
		lo := fromHexChar(src[2*i+1])
		if hi == 255 || lo == 255 {
			return dst, fmt.Errorf("invalid hex character")
		}
		dst[i] = (hi << 4) | lo
	}
	return dst, nil
}

func parseTargetHash(r *http.Request) ([32]byte, error) {
	rawQuery := r.URL.RawQuery
	if strings.HasPrefix(rawQuery, "word=") {
		word := rawQuery[5:]
		if idx := strings.IndexByte(word, '&'); idx != -1 {
			word = word[:idx]
		}
		if word != "" {
			return pkg.GetHash(word), nil
		}
	} else if strings.HasPrefix(rawQuery, "hash=") {
		hashHex := rawQuery[5:]
		if idx := strings.IndexByte(hashHex, '&'); idx != -1 {
			hashHex = hashHex[:idx]
		}
		if hashHex != "" {
			return decodeHex32(hashHex)
		}
	}

	word := r.URL.Query().Get("word")
	if word != "" {
		return pkg.GetHash(word), nil
	}

	hashHex := r.URL.Query().Get("hash")
	if hashHex != "" {
		return decodeHex32(hashHex)
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
		if ok {
			var buf [64]byte
			n := copy(buf[:], `{"word":"`)
			n += copy(buf[n:], word)
			n += copy(buf[n:], `","found":true}`+"\n")
			w.Write(buf[:n])
		} else {
			io.WriteString(w, `{"found":false}`+"\n")
		}
	})

	addr := fmt.Sprintf(":%d", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Erreur serveur : %v", err)
	}
}
