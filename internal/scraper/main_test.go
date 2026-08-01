package scraper

import (
	"os"
	"testing"

	"github.com/standard-user/cinder/pkg/logger"
)

// TestMain initializes the shared logger so real scraper callbacks
// (e.g. colly OnRequest) don't panic on a nil logger.Log.
func TestMain(m *testing.M) {
	logger.Init("error")
	os.Exit(m.Run())
}
