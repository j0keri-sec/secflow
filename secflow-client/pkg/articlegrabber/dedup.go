// Package articlegrabber provides article crawlers for security news and articles.
package articlegrabber

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
)

// Deduplicator provides thread-safe deduplication for articles based on URL and content hash.
type Deduplicator struct {
	mu       sync.RWMutex
	seenURLs map[string]bool
	seenHash map[string]bool
}

// NewDeduplicator creates a new Deduplicator.
func NewDeduplicator() *Deduplicator {
	return &Deduplicator{
		seenURLs: make(map[string]bool),
		seenHash: make(map[string]bool),
	}
}

// IsDuplicate checks if an article is a duplicate.
// Returns true if the article's URL or content hash has been seen before.
func (d *Deduplicator) IsDuplicate(article *Article) bool {
	if article == nil {
		return true
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check URL-based deduplication first (faster)
	if article.URL != "" {
		normalizedURL := normalizeURL(article.URL)
		if d.seenURLs[normalizedURL] {
			return true
		}
	}

	// Check content hash-based deduplication
	contentHash := d.hashContent(article.Title + article.Content)
	if d.seenHash[contentHash] {
		return true
	}

	return false
}

// Add marks an article as seen by its URL and content hash.
func (d *Deduplicator) Add(article *Article) {
	if article == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Add URL if present
	if article.URL != "" {
		d.seenURLs[normalizeURL(article.URL)] = true
	}

	// Add content hash
	contentHash := d.hashContent(article.Title + article.Content)
	d.seenHash[contentHash] = true
}

// AddURL marks a URL as seen.
func (d *Deduplicator) AddURL(url string) {
	if url == "" {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.seenURLs[normalizeURL(url)] = true
}

// AddHash marks a content hash as seen.
func (d *Deduplicator) AddHash(hash string) {
	if hash == "" {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.seenHash[hash] = true
}

// HasURL checks if a URL has been seen before.
func (d *Deduplicator) HasURL(url string) bool {
	if url == "" {
		return false
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.seenURLs[normalizeURL(url)]
}

// HasHash checks if a content hash has been seen before.
func (d *Deduplicator) HasHash(hash string) bool {
	if hash == "" {
		return false
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.seenHash[hash]
}

// Clear removes all tracked URLs and hashes.
func (d *Deduplicator) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.seenURLs = make(map[string]bool)
	d.seenHash = make(map[string]bool)
}

// ClearURLs removes all tracked URLs.
func (d *Deduplicator) ClearURLs() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.seenURLs = make(map[string]bool)
}

// ClearHashes removes all tracked content hashes.
func (d *Deduplicator) ClearHashes() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.seenHash = make(map[string]bool)
}

// Stats returns the number of tracked URLs and hashes.
func (d *Deduplicator) Stats() (urls, hashes int) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.seenURLs), len(d.seenHash)
}

// ContentHash computes a SHA-256 hash of the given content.
func (d *Deduplicator) ContentHash(content string) string {
	return d.hashContent(content)
}

// hashContent computes a SHA-256 hash of the given content string.
func (d *Deduplicator) hashContent(content string) string {
	if content == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// normalizeURL normalizes a URL for consistent comparison.
// Converts to lowercase and trims trailing slashes.
func normalizeURL(url string) string {
	if url == "" {
		return ""
	}
	// Trim trailing slash
	url = strings.TrimRight(url, "/")
	// Convert to lowercase for case-insensitive comparison
	url = strings.ToLower(url)
	return url
}

// BatchAdd adds multiple articles to the deduplicator, skipping duplicates.
// Returns only the non-duplicate articles.
func (d *Deduplicator) BatchAdd(articles []*Article) []*Article {
	if articles == nil {
		return nil
	}

	result := make([]*Article, 0, len(articles))

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, article := range articles {
		if article == nil {
			continue
		}

		// Check for duplicates
		isDup := false
		if article.URL != "" {
			if d.seenURLs[normalizeURL(article.URL)] {
				isDup = true
			}
		}

		if !isDup {
			contentHash := d.hashContent(article.Title + article.Content)
			if d.seenHash[contentHash] {
				isDup = true
			}
		}

		if !isDup {
			// Add to seen sets
			if article.URL != "" {
				d.seenURLs[normalizeURL(article.URL)] = true
			}
			d.seenHash[d.hashContent(article.Title+article.Content)] = true
			result = append(result, article)
		}
	}

	return result
}