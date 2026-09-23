package pkg_test

import (
	"crypto/sha256"
	"testing"

	"github.com/Chroq/HashBreaker/pkg"
)

var (
	SinkRaw [pkg.MaxDepth]byte
)

func TestBruteForce(t *testing.T) {
	tests := []struct {
		name  string
		word  [pkg.MaxDepth]byte
		text  string
		depth int
	}{
		{
			name:  "word z3D",
			word:  [pkg.MaxDepth]byte{'z', '3', 'D'},
			text:  "z3D",
			depth: 3,
		},
		{
			name:  "word Sh3n",
			word:  [pkg.MaxDepth]byte{'S', 'h', '3', 'n'},
			text:  "Sh3n",
			depth: 4,
		},
	}

	for _, tt := range tests {
		t.Run("Map_"+tt.name, func(t *testing.T) {
			hash := sha256.Sum256(tt.word[:tt.depth])
			ref := pkg.NewMapReferential(tt.depth)

			rawWord := ref.GetRaw(hash)
			if rawWord != tt.word {
				t.Errorf("GetRaw: Expected %q, got %q", tt.word, string(rawWord[:]))
			}

			str, ok := ref.Get(hash)
			if !ok || str != tt.text {
				t.Errorf("Get: Expected %q, got %q (ok=%v)", tt.text, str, ok)
			}
		})
	}
}

func TestNotFound(t *testing.T) {
	ref := pkg.NewMapReferential(2)

	// Hash of a 3-character word 'z3D' (not in depth 2 referential)
	hash3 := sha256.Sum256([]byte("z3D"))
	got3 := ref.GetRaw(hash3)
	if got3 != ([pkg.MaxDepth]byte{}) {
		t.Errorf("Expected empty [8]byte for missing hash, got %v", got3)
	}

	str, ok := ref.Get(hash3)
	if ok || str != "" {
		t.Errorf("Expected ('', false) for missing hash, got (%q, %v)", str, ok)
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

		b.Run(tt.word+"_GetRaw", func(b *testing.B) {
			b.ReportAllocs()
			var w [pkg.MaxDepth]byte
			for i := 0; i < b.N; i++ {
				w = mapReferential.GetRaw(hash)
			}
			SinkRaw = w
		})
	}
}
