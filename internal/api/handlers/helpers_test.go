package handlers

import (
	"os"
	"testing"

	"github.com/standard-user/cinder/pkg/logger"
)

func TestMain(m *testing.M) {
	// Initialize logger for tests that need it (scrape handler uses logger.Log)
	logger.Init("error") // Use error level to keep test output clean
	os.Exit(m.Run())
}
