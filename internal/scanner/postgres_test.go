package scanner

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPostgresScanner(t *testing.T) {
	scanner := NewPostgresScanner()
	assert.NotNil(t, scanner)
	assert.IsType(t, &PostgresScanner{}, scanner)
}

func TestPostgresScanner_connectionConfig(t *testing.T) {
	config := ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "postgres",
		Password: "password",
		SSLMode:  "",
	}

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "testdb", config.Database)
	assert.Equal(t, "postgres", config.Username)
	assert.Equal(t, "password", config.Password)
	assert.Equal(t, "", config.SSLMode)
}

func TestPostgresScanner_connectionConfig_WithSSL(t *testing.T) {
	config := ConnectionConfig{
		SSLMode: "require",
	}

	assert.Equal(t, "require", config.SSLMode)
}

// TestPostgresScanner_Integration is PostgreSQL scanner integration test
// Requires a local PostgreSQL instance; set POSTGRES_TEST_DSN to enable
func TestPostgresScanner_Integration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("Skipping PostgreSQL integration test: POSTGRES_TEST_DSN not set")
	}

	// Integration test logic will be implemented when a PostgreSQL test instance is available
	// 1. Create scanner
	// 2. Test connection
	// 3. Scan schema
	// 4. Verify results
}
