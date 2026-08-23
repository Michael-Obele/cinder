package handlers

import (
	"os"
	"testing"

	"github.com/standard-user/cinder/internal/safeurl"
	"github.com/standard-user/cinder/pkg/logger"
)

func TestMain(m *testing.M) {
	// Initialize logger for tests that need it (scrape handler uses logger.Log)
	logger.Init("error") // Use error level to keep test output clean
	// httptest servers bind to 127.0.0.1, which the SSRF guard blocks by
	// design; the map handler fetches through it.
	os.Setenv(safeurl.AllowPrivateEnv, "true")
	os.Exit(m.Run())
}
