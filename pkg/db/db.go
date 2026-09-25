package db

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Chroq/HashBreaker/pkg"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Config holds PostgreSQL connection parameters.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// LoadConfig loads credentials from .env and environment variables.
func LoadConfig() Config {
	_ = godotenv.Load(".env")

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "hashbreaker")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return Config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		DBName:   dbname,
		SSLMode:  sslmode,
	}
}

func (c Config) ConnString() string {
	if c.Password != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
	}
	return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s",
		c.User, c.Host, c.Port, c.DBName, c.SSLMode)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// Connect initializes a connection pool to PostgreSQL.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	connConfig, err := pgxpool.ParseConfig(cfg.ConnString())
	if err != nil {
		return nil, fmt.Errorf("invalid db config: %w", err)
	}

	// Performance tuning for pool
	connConfig.MaxConns = 25
	connConfig.MinConns = 5
	connConfig.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}

// Migrate applies the schema migration (creates table and primary key b-tree index).
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS hashes (
		hash BYTEA PRIMARY KEY,
		word VARCHAR(8) NOT NULL
	);
	`
	_, err := pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}

// combinationCopySource implements pgx.CopyFromSource for high-speed streaming batch insert.
type combinationCopySource struct {
	maxDepth     int
	currentDepth int
	indices      [pkg.MaxDepth]int
	currHash     [32]byte
	currWord     string
	buf          [pkg.MaxDepth]byte
	finished     bool
	err          error
	count        int64
}

func newCombinationCopySource(depth int) *combinationCopySource {
	return &combinationCopySource{
		maxDepth:     depth,
		currentDepth: 1,
	}
}

func (s *combinationCopySource) Next() bool {
	if s.finished || s.currentDepth > s.maxDepth {
		return false
	}

	// Construct current word
	charset := pkg.Charset
	nCharset := len(charset)
	for i := 0; i < s.currentDepth; i++ {
		s.buf[i] = charset[s.indices[i]]
	}
	s.currWord = string(s.buf[:s.currentDepth])
	s.currHash = sha256.Sum256(s.buf[:s.currentDepth])
	s.count++

	// Advance indices for next iteration
	pos := s.currentDepth - 1
	for pos >= 0 {
		s.indices[pos]++
		if s.indices[pos] < nCharset {
			break
		}
		s.indices[pos] = 0
		pos--
	}

	if pos < 0 {
		// Completed current depth, move to next depth
		s.currentDepth++
		for i := range s.indices {
			s.indices[i] = 0
		}
	}

	return true
}

func (s *combinationCopySource) Values() ([]any, error) {
	return []any{s.currHash[:], s.currWord}, nil
}

func (s *combinationCopySource) Err() error {
	return s.err
}

// BatchInsertDepth populates the database with all combinations up to maxDepth using PostgreSQL COPY.
func BatchInsertDepth(ctx context.Context, pool *pgxpool.Pool, depth int) (int64, error) {
	// Check if already populated
	var currentCount int64
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM hashes").Scan(&currentCount)
	if err != nil {
		return 0, fmt.Errorf("failed to count existing rows: %w", err)
	}

	expected := int64(pkg.TotalCombinations(depth))
	if currentCount >= expected {
		log.Printf("ℹ️  Table 'hashes' déjà peuplée avec %d lignes (attendu: %d). Insertion ignorée.", currentCount, expected)
		return currentCount, nil
	}

	log.Printf("🚀 Début du batch insert (COPY) pour profondeur %d (%d combinaisons)...", depth, expected)
	start := time.Now()

	// Truncate to ensure clean state
	if _, err := pool.Exec(ctx, "TRUNCATE TABLE hashes"); err != nil {
		return 0, fmt.Errorf("failed to truncate hashes table: %w", err)
	}

	source := newCombinationCopySource(depth)
	rowsAffected, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"hashes"},
		[]string{"hash", "word"},
		source,
	)
	if err != nil {
		return 0, fmt.Errorf("COPY failed: %w", err)
	}

	elapsed := time.Since(start)
	log.Printf("✅ Batch insert terminé : %d lignes insérées en %v (%.0f rows/sec)",
		rowsAffected, elapsed, float64(rowsAffected)/elapsed.Seconds())

	return rowsAffected, nil
}

// GetWordByHash performs a B-Tree indexed lookup of a 32-byte hash in PostgreSQL.
func GetWordByHash(ctx context.Context, pool *pgxpool.Pool, hash [32]byte) (string, error) {
	var word string
	err := pool.QueryRow(ctx, "SELECT word FROM hashes WHERE hash = $1", hash[:]).Scan(&word)
	if err != nil {
		return "", err
	}
	return word, nil
}
