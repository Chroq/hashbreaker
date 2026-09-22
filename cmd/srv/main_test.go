package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chroq/HashBreaker/pkg"
)

func BenchmarkHandler(b *testing.B) {
	mapRef := pkg.NewMapReferential(3)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			ioWriteString(w, `{"found":false}`+"\n")
		}
	})

	b.Run("WordParam", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/guess?word=z3D", nil)
		rec := httptest.NewRecorder()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rec.Body.Reset()
			handler.ServeHTTP(rec, req)
		}
	})

	b.Run("HashHexParam", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/guess?hash=a532ca5e11e2b06ccc911e0d962a4864cdb87da05723f3a050a376d0f0895e63", nil)
		rec := httptest.NewRecorder()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rec.Body.Reset()
			handler.ServeHTTP(rec, req)
		}
	})
}

func ioWriteString(w http.ResponseWriter, s string) {
	w.Write([]byte(s))
}
