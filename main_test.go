package main_test

import (
	"crypto/sha256"
	"testing"

	main "github.com/Chroq/HashBreaker"
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
		t.Run("Iterative_"+tt.name, func(t *testing.T) {
			hash := sha256.Sum256([]byte(tt.word))
			got := main.NewArrayReferential(tt.depth).Get(hash)

			if got != tt.word {
				t.Errorf("Expected %q, got %q", tt.word, got)
			}
		})
	}
}

func BenchmarkNewArrayReferential(b *testing.B) {
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
			main.NewArrayReferential(tt.depth)
		})

		b.Run(tt.word+"_MapReferential", func(b *testing.B) {
			main.NewMapReferential(tt.depth)
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
		arrayReferential := main.NewArrayReferential(tt.depth)
		mapReferential := main.NewMapReferential(tt.depth)

		b.Run(tt.word+"_ArrayReferential", func(b *testing.B) {
			arrayReferential.Get(sha256.Sum256([]byte(tt.word)))
		})

		b.Run(tt.word+"_MapReferential", func(b *testing.B) {
			mapReferential.Get(sha256.Sum256([]byte(tt.word)))
		})
	}
}
