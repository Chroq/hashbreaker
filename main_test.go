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
		t.Run("Recursive_"+tt.name, func(t *testing.T) {
			hash := sha256.Sum256([]byte(tt.word))
			got := main.BruteForceRecursive(hash, tt.depth)

			if got != tt.word {
				t.Errorf("Expected %q, got %q", tt.word, got)
			}
		})

		t.Run("Iterative_"+tt.name, func(t *testing.T) {
			hash := sha256.Sum256([]byte(tt.word))
			got := main.BruteForceIterative(hash, tt.depth)

			if got != tt.word {
				t.Errorf("Expected %q, got %q", tt.word, got)
			}
		})
	}
}

func TestBruteForceRecursive(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			hash := sha256.Sum256([]byte(tt.word))
			got := main.BruteForceRecursive(hash, tt.depth)

			if got != tt.word {
				t.Errorf("Expected %q, got %q", tt.word, got)
			}
		})
	}
}

func BenchmarkBruteForce(b *testing.B) {
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
		b.Run(tt.word+"_Recursive", func(b *testing.B) {
			hash := sha256.Sum256([]byte(tt.word))
			main.BruteForceRecursive(hash, tt.depth)
		})

		b.Run(tt.word+"_Iterative", func(b *testing.B) {
			hash := sha256.Sum256([]byte(tt.word))
			main.BruteForceIterative(hash, tt.depth)
		})
	}
}
