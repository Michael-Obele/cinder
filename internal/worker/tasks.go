package worker

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeScrape = "scrape:url"
)

type ScrapePayload struct {
	URL    string `json:"url"`
	Render bool   `json:"render"`
}

// NewScrapeTask creates a new task for scraping a URL.
func NewScrapeTask(url string, render bool) (*asynq.Task, error) {
	payload := ScrapePayload{
		URL:    url,
		Render: render,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scrape payload: %w", err)
	}
	return asynq.NewTask(TypeScrape, data), nil
}
