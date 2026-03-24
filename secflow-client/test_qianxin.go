// +build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/secflow/client/pkg/articlegrabber"
)

func main() {
	fmt.Println("=== Testing Qianxin Weekly Grabber ===")

	logDir := "./log"
	os.MkdirAll(logDir, 0755)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Get the grabber
	g, ok := articlegrabber.Get("qianxin-weekly")
	if !ok {
		fmt.Println("ERROR: qianxin-weekly grabber not found")
		return
	}

	fmt.Printf("Grabber name: %s\n", g.Name())

	// Fetch articles
	articles, err := g.Fetch(ctx, 3)
	if err != nil {
		fmt.Printf("ERROR: Fetch failed: %v\n", err)
		return
	}

	fmt.Printf("\n=== Fetched %d articles ===\n\n", len(articles))

	timestamp := time.Now().Format("20060102_150405")

	for i, a := range articles {
		fmt.Printf("--- Article %d ---\n", i+1)
		fmt.Printf("Title: %s\n", a.Title)
		fmt.Printf("Author: %s\n", a.Author)
		fmt.Printf("Source: %s\n", a.Source)
		fmt.Printf("URL: %s\n", a.URL)
		fmt.Printf("PublishedAt: %v\n", a.PublishedAt)
		fmt.Printf("Tags: %v\n", a.Tags)
		fmt.Printf("\nSummary:\n%s\n", a.Summary)
		fmt.Printf("\nContent (%d chars):\n%s\n", len(a.Content), a.Content)

		// Save article to log file
		filename := filepath.Join(logDir, fmt.Sprintf("article_%d_%s.txt", i+1, timestamp))
		f, err := os.Create(filename)
		if err != nil {
			fmt.Printf("ERROR: Failed to create log file: %v\n", err)
			continue
		}

		// Write formatted output
		fmt.Fprintf(f, "=== Article %d ===\n", i+1)
		fmt.Fprintf(f, "Title: %s\n", a.Title)
		fmt.Fprintf(f, "Author: %s\n", a.Author)
		fmt.Fprintf(f, "Source: %s\n", a.Source)
		fmt.Fprintf(f, "URL: %s\n", a.URL)
		fmt.Fprintf(f, "PublishedAt: %v\n", a.PublishedAt)
		fmt.Fprintf(f, "Tags: %v\n", a.Tags)
		fmt.Fprintf(f, "\n=== Summary ===\n%s\n", a.Summary)
		fmt.Fprintf(f, "\n=== Content (%d chars) ===\n%s\n", len(a.Content), a.Content)

		f.Close()
		fmt.Printf("Saved to: %s\n\n", filename)

		// Pretty print the article as JSON
		data, _ := json.MarshalIndent(a, "", "  ")
		fmt.Printf("\nFull JSON:\n%s\n", string(data))
		fmt.Println()
	}

	// Save all articles as JSON array
	jsonFilename := filepath.Join(logDir, fmt.Sprintf("articles_%s.json", timestamp))
	jsonData, _ := json.MarshalIndent(articles, "", "  ")
	os.WriteFile(jsonFilename, jsonData, 0644)
	fmt.Printf("All articles saved to: %s\n", jsonFilename)
}