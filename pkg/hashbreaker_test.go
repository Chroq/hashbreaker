package pkg_test

import (
	"crypto/sha256"
	"testing"

	"github.com/Chroq/HashBreaker/pkg"
)

var (
	SinkString string
	SinkRaw    [pkg.MaxDepth]byte
	SinkLen    int
	SinkOk     bool
)

func TestBruteForce(t *testing.T) {
	tests := []struct {
		name  string
		word  string
		depth int
	}{
		{
			name:  "word z3D",
			word:  "z3D",
			depth: 3,
		},
		{
			name:  "word Sh3n",
			word:  "Sh3n",
			depth: 4,
		},
	}

	for _, tt := range tests {
		t.Run("Map_"+tt.name, func(t *testing.T) {
			hash := sha256.Sum256([]byte(tt.word))
			ref := pkg.NewMapReferential(tt.depth)

			got, ok := ref.Get(hash)
			if !ok || got != tt.word {
				t.Errorf("Get: Expected %q, got %q", tt.word, got)
			}

			rawWord, n, okRaw := ref.GetRaw(hash)
			if !okRaw || string(rawWord[:n]) != tt.word {
				t.Errorf("GetRaw: Expected %q, got %q", tt.word, string(rawWord[:n]))
			}
		})
	}
}

func BenchmarkNewReferential(b *testing.B) {
	targets := []struct {
		word  string
		depth int
	}{
		{
			word:  "z3D",
			depth: 3,
		},
		{
			word:  "Sh3n",
			depth: 4,
		},
	}

	for _, tt := range targets {
		b.Run(tt.word+"_MapReferential", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = pkg.NewMapReferential(tt.depth)
			}
		})
	}
}

func BenchmarkGet(b *testing.B) {
	targets := []struct {
		word  string
		depth int
	}{
		{
			word:  "z3D",
			depth: 3,
		},
		{
			word:  "Sh3n",
			depth: 4,
		},
	}

	for _, tt := range targets {
		mapReferential := pkg.NewMapReferential(tt.depth)
		hash := sha256.Sum256([]byte(tt.word))

		// Benchmark Get standard avec consommation du résultat (anti-DCE)
		b.Run(tt.word+"_MapReferential", func(b *testing.B) {
			b.ReportAllocs()
			var s string
			var ok bool
			for i := 0; i < b.N; i++ {
				s, ok = mapReferential.Get(hash)
			}
			SinkString = s
			SinkOk = ok
		})

		// Benchmark GetRaw zéro-allocation natif
		b.Run(tt.word+"_GetRaw", func(b *testing.B) {
			b.ReportAllocs()
			var w [pkg.MaxDepth]byte
			var n int
			var ok bool
			for i := 0; i < b.N; i++ {
				w, n, ok = mapReferential.GetRaw(hash)
			}
			SinkRaw = w
			SinkLen = n
			SinkOk = ok
		})
	}
}
