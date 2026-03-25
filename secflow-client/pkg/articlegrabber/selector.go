// Package articlegrabber provides article crawlers for security news and articles.
package articlegrabber

import (
	"errors"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ErrNoContentFound is returned when no content is found with any selector.
var ErrNoContentFound = errors.New("no content found with any selector")

// SmartSelector provides CSS selector fallback for content extraction.
type SmartSelector struct {
	selectors []string
	content   string
}

// NewSmartSelector creates a new SmartSelector with the given priority-ordered CSS selectors.
func NewSmartSelector(selectors []string) *SmartSelector {
	return &SmartSelector{
		selectors: selectors,
	}
}

// NewSmartSelectorFromContent creates a new SmartSelector with content already loaded.
func NewSmartSelectorFromContent(selectors []string, content string) *SmartSelector {
	return &SmartSelector{
		selectors: selectors,
		content:   content,
	}
}

// Extract extracts content using the first selector that returns non-empty results.
// Returns the extracted HTML content or an error if no selector matched.
func (s *SmartSelector) Extract() (string, error) {
	if s.content == "" {
		return "", ErrNoContentFound
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s.content))
	if err != nil {
		return "", err
	}

	return s.TrySelectorsFromDoc(doc)
}

// TrySelectors iterates through all selectors and returns content from the first match.
// Returns the extracted HTML content or an error if no selector matched.
func (s *SmartSelector) TrySelectors() (string, error) {
	if s.content == "" {
		return "", ErrNoContentFound
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s.content))
	if err != nil {
		return "", err
	}

	return s.TrySelectorsFromDoc(doc)
}

// TrySelectorsFromDoc tries each selector in order on the provided document.
// Returns content from the first selector that finds a match with sufficient content.
func (s *SmartSelector) TrySelectorsFromDoc(doc *goquery.Document) (string, error) {
	if doc == nil {
		return "", ErrNoContentFound
	}

	for _, selector := range s.selectors {
		content := s.extractWithSelector(doc, selector)
		if content != "" {
			return content, nil
		}
	}

	return "", ErrNoContentFound
}

// extractWithSelector extracts content using a single CSS selector.
// Returns empty string if selector doesn't match or content is insufficient.
func (s *SmartSelector) extractWithSelector(doc *goquery.Document, selector string) string {
	selection := doc.Find(selector)
	if selection.Length() == 0 {
		return ""
	}

	// Get HTML content
	html, err := selection.Html()
	if err != nil || html == "" {
		// Try text content as fallback
		text := selection.Text()
		if len(strings.TrimSpace(text)) > 100 {
			return text
		}
		return ""
	}

	// Check if content is substantial (at least 100 characters)
	if len(html) < 100 {
		text := selection.Text()
		if len(strings.TrimSpace(text)) > 100 {
			return text
		}
		return ""
	}

	return html
}

// SetContent sets the HTML content to extract from.
func (s *SmartSelector) SetContent(content string) {
	s.content = content
}

// SetSelectors updates the priority-ordered CSS selectors.
func (s *SmartSelector) SetSelectors(selectors []string) {
	s.selectors = selectors
}

// AddSelector appends a selector to the end of the priority list.
func (s *SmartSelector) AddSelector(selector string) {
	s.selectors = append(s.selectors, selector)
}

// PrependSelector adds a selector at the beginning of the priority list.
func (s *SmartSelector) PrependSelector(selector string) {
	s.selectors = append([]string{selector}, s.selectors...)
}

// DefaultSelectors provides common selectors for different article sources.
var DefaultSelectors = map[string][]string{
	// Xianzhi (先知社区) selectors
	"xianzhi": {
		".article-content",
		".post-content",
		".article-body",
		"#content",
		".main-content",
		"article",
		".container",
	},

	// Qianxin (奇安信) selectors
	"qianxin": {
		"#poc-preview",
		"#poc-preview .content",
		".article .content",
		".article-content",
		".notice-detail .content",
		".detail-container .article",
	},

	// Venustech (启明星辰) selectors
	"venustech": {
		"div.news_text",
		".news-content",
		".article-detail",
		".content-body",
		"article",
	},

	// Sihou (嘶吼) selectors
	"sihou": {
		".article-content",
		".post-content",
		".article-body",
		".entry-content",
		"article",
	},

	// FreeBuf selectors
	"freebuf": {
		".article-content",
		".post-content",
		".content-body",
		"article",
	},

	// SecurityWeek selectors
	"securityweek": {
		".article-body",
		".post-content",
		".entry-content",
		"article",
	},

	// Generic fallback selectors
	"generic": {
		".article-content",
		".post-content",
		".article-body",
		".content-body",
		".entry-content",
		"article",
		"main",
		".main-content",
		"#content",
		".container",
	},
}

// GetSelectorsForSource returns the default selectors for a known source.
func GetSelectorsForSource(source string) []string {
	if selectors, ok := DefaultSelectors[strings.ToLower(source)]; ok {
		return selectors
	}
	return DefaultSelectors["generic"]
}

// SelectorResult holds the result of a selector extraction attempt.
type SelectorResult struct {
	Selector string
	Content  string
	Length   int
	Found    bool
}

// TryAllSelectors attempts extraction with all selectors and returns results for each.
// Useful for debugging which selectors work best.
func (s *SmartSelector) TryAllSelectors() []SelectorResult {
	results := make([]SelectorResult, 0, len(s.selectors))

	if s.content == "" {
		return results
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s.content))
	if err != nil {
		return results
	}

	for _, selector := range s.selectors {
		result := SelectorResult{
			Selector: selector,
		}

		selection := doc.Find(selector)
		if selection.Length() == 0 {
			result.Found = false
			results = append(results, result)
			continue
		}

		html, _ := selection.Html()
		text := selection.Text()

		// Prefer HTML, fall back to text
		if len(html) >= 100 {
			result.Content = html
			result.Length = len(html)
			result.Found = true
		} else if len(strings.TrimSpace(text)) >= 100 {
			result.Content = strings.TrimSpace(text)
			result.Length = len(result.Content)
			result.Found = true
		} else {
			result.Found = false
		}

		results = append(results, result)
	}

	return results
}

// ExtractText extracts plain text using the first successful selector.
func (s *SmartSelector) ExtractText() (string, error) {
	if s.content == "" {
		return "", ErrNoContentFound
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s.content))
	if err != nil {
		return "", err
	}

	for _, selector := range s.selectors {
		selection := doc.Find(selector)
		if selection.Length() == 0 {
			continue
		}

		text := selection.Text()
		if len(strings.TrimSpace(text)) >= 100 {
			return strings.TrimSpace(text), nil
		}
	}

	return "", ErrNoContentFound
}