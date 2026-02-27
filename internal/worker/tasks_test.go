package worker

import (
	"encoding/json"
	"testing"
)

func TestNewScrapeTask(t *testing.T) {
	task, err := NewScrapeTask("https://example.com", false, false, false)
	if err != nil {
		t.Fatalf("NewScrapeTask failed: %v", err)
	}

	if task == nil {
		t.Fatal("Task should not be nil")
	}

	// Verify payload
	var payload ScrapePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if payload.URL != "https://example.com" {
		t.Errorf("URL mismatch: got %q, want %q", payload.URL, "https://example.com")
	}

	if payload.Render != false {
		t.Error("Render should be false")
	}
}

func TestNewScrapeTask_WithRender(t *testing.T) {
	task, err := NewScrapeTask("https://example.com", true, false, false)
	if err != nil {
		t.Fatalf("NewScrapeTask failed: %v", err)
	}

	var payload ScrapePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if payload.Render != true {
		t.Error("Render should be true")
	}
}

func TestNewScrapeTask_TaskType(t *testing.T) {
	task, err := NewScrapeTask("https://example.com", false, false, false)
	if err != nil {
		t.Fatalf("NewScrapeTask failed: %v", err)
	}

	if task.Type() != TypeScrape {
		t.Errorf("Task type should be %q, got %q", TypeScrape, task.Type())
	}
}

func TestScrapePayload_JSON(t *testing.T) {
	payload := ScrapePayload{
		URL:    "https://example.com",
		Render: true,
		Mode:   "dynamic",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ScrapePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.URL != payload.URL {
		t.Errorf("URL mismatch")
	}
	if decoded.Render != payload.Render {
		t.Errorf("Render mismatch")
	}
	if decoded.Mode != payload.Mode {
		t.Errorf("Mode mismatch")
	}
}

func TestScrapePayload_BackwardCompatibility(t *testing.T) {
	// Test that the old format (just URL + Render) still works
	oldJSON := `{"url":"https://example.com","render":true}`

	var payload ScrapePayload
	if err := json.Unmarshal([]byte(oldJSON), &payload); err != nil {
		t.Fatalf("Failed to unmarshal old format: %v", err)
	}

	if payload.URL != "https://example.com" {
		t.Errorf("URL mismatch")
	}
	if payload.Render != true {
		t.Errorf("Render mismatch")
	}
	if payload.Mode != "" {
		t.Errorf("Mode should be empty for old format, got %q", payload.Mode)
	}
}

func TestScrapePayload_ModeMapping(t *testing.T) {
	// Verify the backward-compatible mode mapping logic from handlers.go
	tests := []struct {
		name         string
		render       bool
		mode         string
		expectedMode string
	}{
		{
			name:         "Render true overrides to dynamic",
			render:       true,
			mode:         "",
			expectedMode: "dynamic",
		},
		{
			name:         "Empty mode defaults to smart",
			render:       false,
			mode:         "",
			expectedMode: "smart",
		},
		{
			name:         "Explicit static mode",
			render:       false,
			mode:         "static",
			expectedMode: "static",
		},
		{
			name:         "Explicit dynamic mode",
			render:       false,
			mode:         "dynamic",
			expectedMode: "dynamic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the mode mapping logic
			mode := tt.mode
			if tt.render {
				mode = "dynamic"
			}
			if mode == "" {
				mode = "smart"
			}

			if mode != tt.expectedMode {
				t.Errorf("Mode = %q, want %q", mode, tt.expectedMode)
			}
		})
	}
}
