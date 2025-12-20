package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	baseURL = "http://localhost:8080"
)

type ScrapeRequest struct {
	URL    string `json:"url"`
	Render bool   `json:"render"`
}

type CrawlResponse struct {
	ID string `json:"id"`
}

type CrawlStatusResponse struct {
	ID     string      `json:"id"`
	Status string      `json:"state"`
	Result interface{} `json:"result,omitempty"`
}

func main() {
	outputDir := flag.String("output", "test_reports/new_run", "Directory to store test results")
	flag.Parse()

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	report := "# E2E Test Report\n\n"
	report += fmt.Sprintf("Date: %s\n\n", time.Now().Format(time.RFC1123))

	// Test 1: Static Scrape
	fmt.Println("Running Static Scrape Test...")
	staticRes, err := runScrapeTest("https://httpbin.org/html", false, filepath.Join(*outputDir, "static.json"))
	if err != nil {
		report += "## ❌ Static Scrape Test Failed\n\n"
		report += fmt.Sprintf("Error: %v\n\n", err)
	} else {
		report += "## ✅ Static Scrape Test Passed\n\n"
		report += fmt.Sprintf("Saved to: `static.json`\n")
		report += fmt.Sprintf("Response size: %d bytes\n\n", len(staticRes))
	}

	// Test 2: Dynamic Scrape (using example.com as it's light, but render=true triggers chromedp)
	fmt.Println("Running Dynamic Scrape Test...")
	dynamicRes, err := runScrapeTest("https://example.com", true, filepath.Join(*outputDir, "dynamic.json"))
	if err != nil {
		report += "## ❌ Dynamic Scrape Test Failed\n\n"
		report += fmt.Sprintf("Error: %v\n\n", err)
	} else {
		report += "## ✅ Dynamic Scrape Test Passed\n\n"
		report += fmt.Sprintf("Saved to: `dynamic.json`\n")
		report += fmt.Sprintf("Response size: %d bytes\n\n", len(dynamicRes))
	}

	// Test 3: Async Crawl
	fmt.Println("Running Async Crawl Test...")
	crawlRes, err := runCrawlTest("https://httpbin.org/html", filepath.Join(*outputDir, "crawl.json"))
	if err != nil {
		report += "## ❌ Async Crawl Test Failed\n\n"
		report += fmt.Sprintf("Error: %v\n\n", err)
	} else {
		report += "## ✅ Async Crawl Test Passed\n\n"
		report += fmt.Sprintf("Saved to: `crawl.json`\n")
		report += fmt.Sprintf("Final Status: %s\n\n", crawlRes)
	}

	// Save Report
	reportPath := filepath.Join(*outputDir, "TEST_REPORT.md")
	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		fmt.Printf("Error writing report: %v\n", err)
	}
	fmt.Printf("Test run complete. Report saved to %s\n", reportPath)
}

func runScrapeTest(targetURL string, render bool, outputPath string) ([]byte, error) {
	reqBody, _ := json.Marshal(ScrapeRequest{URL: targetURL, Render: render})
	resp, err := http.Post(baseURL+"/v1/scrape", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s, body: %s", resp.Status, string(body))
	}

	if err := os.WriteFile(outputPath, body, 0644); err != nil {
		return nil, err
	}

	return body, nil
}

func runCrawlTest(targetURL string, outputPath string) (string, error) {
	// 1. Enqueue
	reqBody, _ := json.Marshal(ScrapeRequest{URL: targetURL, Render: false})
	resp, err := http.Post(baseURL+"/v1/crawl", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 { // Accepted
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to enqueue: %s, body: %s", resp.Status, string(body))
	}

	var crawlResp CrawlResponse
	if err := json.NewDecoder(resp.Body).Decode(&crawlResp); err != nil {
		return "", err
	}
	fmt.Printf("Crawl enqueued, ID: %s\n", crawlResp.ID)

	// 2. Poll Status
	maxRetries := 20
	for i := 0; i < maxRetries; i++ {
		time.Sleep(2 * time.Second)
		statusResp, err := http.Get(fmt.Sprintf("%s/v1/crawl/%s", baseURL, crawlResp.ID))
		if err != nil {
			fmt.Printf("Polling error: %v\n", err)
			continue
		}
		
		body, _ := io.ReadAll(statusResp.Body)
		statusResp.Body.Close()

		var status CrawlStatusResponse
		if err := json.Unmarshal(body, &status); err != nil {
			continue
		}

		fmt.Printf("Polling status... %s\n", status.Status)

		if status.Status == "completed" {
			// Save full result
			if err := os.WriteFile(outputPath, body, 0644); err != nil {
				return "", err
			}
			return status.Status, nil
		}
		if status.Status == "failed" {
			return "", fmt.Errorf("task failed remotely")
		}
	}

	return "", fmt.Errorf("timeout waiting for crawling to complete")
}
