package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// SignatureHeader is the header carrying the HMAC-SHA256 webhook signature.
const SignatureHeader = "X-Cinder-Signature"

// webhookAttempts is the number of delivery attempts before giving up.
const webhookAttempts = 3

// webhookTimeout returns the per-attempt HTTP timeout
// (env WEBHOOK_TIMEOUT seconds, default 10s).
func webhookTimeout() time.Duration {
	raw := os.Getenv("WEBHOOK_TIMEOUT")
	if raw == "" {
		return 10 * time.Second
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 10 * time.Second
	}
	return time.Duration(n) * time.Second
}

// Deliver posts payload to webhookURL with an HMAC-SHA256 signature header,
// retrying transient failures up to webhookAttempts times with backoff.
// Delivery failure never fails the crawl — callers treat the error as a
// recorded warning.
func Deliver(ctx context.Context, webhookURL, secret string, payload []byte) error {
	client := &http.Client{Timeout: webhookTimeout()}
	var lastErr error

	for attempt := 1; attempt <= webhookAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("webhook request build failed: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(SignatureHeader, "sha256="+signPayload(secret, payload))

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("webhook delivery attempt %d failed: %w", attempt, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("webhook delivery attempt %d returned status %d", attempt, resp.StatusCode)
		// 4xx means the endpoint rejected the payload — retrying won't help.
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return lastErr
		}
	}
	return lastErr
}

// signPayload returns the hex HMAC-SHA256 of payload keyed by secret.
func signPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
