package db_test

import (
	"context"
	"crypto/sha256"
	"testing"

	"github.com/Chroq/HashBreaker/pkg"
	"github.com/Chroq/HashBreaker/pkg/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPool(t testing.TB) *pgxpool.Pool {
	cfg := db.LoadConfig()
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		t.Skipf("PostgreSQL indisponible : %v", err)
	}
	return pool
}

func TestDBFlow(t *testing.T) {
	ctx := context.Background()
	pool := getTestPool(t)
	defer pool.Close()

	// 1. Migration
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Erreur migration : %v", err)
	}

	// 2. Batch Insert d=3
	count, err := db.BatchInsertDepth(ctx, pool, 3)
	if err != nil {
		t.Fatalf("Erreur batch insert : %v", err)
	}
	expected := int64(pkg.TotalCombinations(3))
	if count < expected {
		t.Errorf("Attendu au moins %d lignes, obtenu %d", expected, count)
	}

	// 3. Retrieval 'z3D'
	targetWord := "z3D"
	targetHash := sha256.Sum256([]byte(targetWord))
	gotWord, err := db.GetWordByHash(ctx, pool, targetHash)
	if err != nil {
		t.Fatalf("Erreur récupération '%s' : %v", targetWord, err)
	}
	if gotWord != targetWord {
		t.Errorf("Attendu '%s', obtenu '%s'", targetWord, gotWord)
	}
}

func BenchmarkPostgresGet_z3D(b *testing.B) {
	ctx := context.Background()
	pool := getTestPool(b)
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		b.Fatalf("Erreur migration : %v", err)
	}
	if _, err := db.BatchInsertDepth(ctx, pool, 3); err != nil {
		b.Fatalf("Erreur batch insert : %v", err)
	}

	targetWord := "z3D"
	targetHash := sha256.Sum256([]byte(targetWord))

	// Warmup
	_, err := db.GetWordByHash(ctx, pool, targetHash)
	if err != nil {
		b.Fatalf("Erreur warmup : %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w, err := db.GetWordByHash(ctx, pool, targetHash)
		if err != nil {
			b.Fatalf("Erreur lookup : %v", err)
		}
		if w != targetWord {
			b.Fatalf("Attendu '%s', obtenu '%s'", targetWord, w)
		}
	}
}

func BenchmarkMemoryVsPostgres_z3D(b *testing.B) {
	ctx := context.Background()
	pool := getTestPool(b)
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		b.Fatalf("Erreur migration : %v", err)
	}
	if _, err := db.BatchInsertDepth(ctx, pool, 3); err != nil {
		b.Fatalf("Erreur batch insert : %v", err)
	}

	targetWord := "z3D"
	targetHash := sha256.Sum256([]byte(targetWord))
	mapRef := pkg.NewMapReferential(3)

	b.Run("InMemory_V5_SoA_GetRaw", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = mapRef.GetRaw(targetHash)
		}
	})

	b.Run("PostgreSQL_BTree_Index_Query", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := db.GetWordByHash(ctx, pool, targetHash)
			if err != nil {
				b.Fatalf("Erreur lookup : %v", err)
			}
		}
	})
}
