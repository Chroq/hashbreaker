package pkg_test

import (
	"crypto/sha256"
	"testing"

	"github.com/Chroq/HashBreaker/pkg"
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
		t.Run("Array_"+tt.name, func(t *testing.T) {
			hash := sha256.Sum256([]byte(tt.word))
			got := pkg.NewArrayReferential(tt.depth).Get(hash)

			if got != tt.word {
				t.Errorf("Expected %q, got %q", tt.word, got)
			}
		})

		t.Run("Map_"+tt.name, func(t *testing.T) {
			hash := sha256.Sum256([]byte(tt.word))
			got, ok := pkg.NewMapReferential(tt.depth).Get(hash)

			if !ok || got != tt.word {
				t.Errorf("Expected %q, got %q", tt.word, got)
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
		b.Run(tt.word+"_ArrayReferential", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				pkg.NewArrayReferential(tt.depth)
			}
		})

		b.Run(tt.word+"_MapReferential", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				pkg.NewMapReferential(tt.depth)
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
		arrayReferential := pkg.NewArrayReferential(tt.depth)
		mapReferential := pkg.NewMapReferential(tt.depth)
		hash := sha256.Sum256([]byte(tt.word))

		b.Run(tt.word+"_ArrayReferential", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				arrayReferential.Get(hash)
			}
		})

		b.Run(tt.word+"_MapReferential", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				mapReferential.Get(hash)
			}
		})
	}
}
