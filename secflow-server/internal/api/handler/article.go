package handler

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/repository"
)

// isPrivateURL checks if a URL points to a private/internal network.
// This prevents SSRF attacks that target internal services.
func isPrivateURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true // Treat unparseable URLs as private
	}

	// Only allow HTTPS
	if u.Scheme != "https" {
		return true
	}

	host := u.Hostname()
	if host == "" {
		return true
	}

	// Check for IP addresses
	ip := net.ParseIP(host)
	if ip != nil {
		// Block private, loopback, and link-local addresses
		return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
	}

	// Block known internal hostnames
	lowerHost := strings.ToLower(host)
	if strings.HasSuffix(lowerHost, ".internal") ||
		strings.HasSuffix(lowerHost, ".local") ||
		strings.HasSuffix(lowerHost, ".localhost") ||
		lowerHost == "localhost" ||
		strings.HasPrefix(lowerHost, "metadata.google.internal") ||
		strings.HasPrefix(lowerHost, "169.254.169.254") {
		return true
	}

	return false
}

// ArticleHandler handles article-related HTTP endpoints.
type ArticleHandler struct {
	repo *repository.ArticleRepository
}

// NewArticleHandler creates a new ArticleHandler.
func NewArticleHandler(repo *repository.ArticleRepository) *ArticleHandler {
	return &ArticleHandler{repo: repo}
}

// List godoc
//
//	GET /api/v1/articles
//	Query params: page, page_size, keyword, source, pushed
func (h *ArticleHandler) List(c *gin.Context) {
	page, pageSize := paginate(c)

	filter := bson.D{}
	if kw := c.Query("keyword"); kw != "" {
		filter = append(filter, bson.E{Key: "title", Value: bson.D{{Key: "$regex", Value: kw}, {Key: "$options", Value: "i"}}})
	}
	if src := c.Query("source"); src != "" {
		filter = append(filter, bson.E{Key: "source", Value: src})
	}
	if pushed := c.Query("pushed"); pushed == "true" {
		filter = append(filter, bson.E{Key: "pushed", Value: true})
	} else if pushed == "false" {
		filter = append(filter, bson.E{Key: "pushed", Value: false})
	}

	items, total, err := h.repo.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		log.Error().Err(err).Msg("list articles")
		Err(c, http.StatusInternalServerError, "database error")
		return
	}
	OK(c, PageResult(items, total, page, pageSize))
}

// Get godoc
//
//	GET /api/v1/articles/:id
func (h *ArticleHandler) Get(c *gin.Context) {
	oid, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Err(c, http.StatusBadRequest, "invalid id")
		return
	}
	article, err := h.repo.GetByID(c.Request.Context(), oid)
	if err != nil {
		Err(c, http.StatusNotFound, "not found")
		return
	}
	OK(c, article)
}

// Delete godoc
//
//	DELETE /api/v1/articles/:id
func (h *ArticleHandler) Delete(c *gin.Context) {
	oid, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Err(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), oid); err != nil {
		log.Error().Err(err).Msg("delete article")
		Err(c, http.StatusInternalServerError, "delete failed")
		return
	}
	OK(c, nil)
}

// ── Upsert (called by node upload) ────────────────────────────────────────

// ArticleUpsertRequest is the payload accepted when a node uploads articles.
type ArticleUpsertRequest struct {
	Items []*model.Article `json:"items"`
}

// Upsert godoc
//
//	POST /api/v1/articles/upsert
func (h *ArticleHandler) Upsert(c *gin.Context) {
	var req ArticleUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Err(c, http.StatusBadRequest, err.Error())
		return
	}
	count, err := h.repo.BulkUpsert(c.Request.Context(), req.Items)
	if err != nil {
		log.Error().Err(err).Msg("upsert articles")
		Err(c, http.StatusInternalServerError, "upsert failed")
		return
	}
	OK(c, gin.H{"upserted": count})
}

// ── Image Upload ──────────────────────────────────────────────────────────────

// UploadImageFromURL downloads an image from a remote URL and saves it locally.
// Returns the local URL path.
//
//	POST /api/v1/articles/upload-image
type ImageUploadRequest struct {
	ImageURL string `json:"image_url" binding:"required,url"`
}

func (h *ArticleHandler) UploadImageFromURL(c *gin.Context) {
	var req ImageUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Err(c, http.StatusBadRequest, "image_url is required and must be a valid URL: "+err.Error())
		return
	}

	// SSRF protection: reject private/internal URLs
	if isPrivateURL(req.ImageURL) {
		log.Warn().Str("url", req.ImageURL).Msg("SSRF attempt blocked: private URL not allowed")
		Err(c, http.StatusBadRequest, "image URL must be a public HTTPS URL")
		return
	}

	// Create uploads directory if it doesn't exist
	uploadDir := "./uploads/images"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Error().Err(err).Msg("failed to create upload directory")
		Err(c, http.StatusInternalServerError, "failed to create upload directory")
		return
	}

	// Download the image
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(req.ImageURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to download image")
		Err(c, http.StatusBadGateway, "failed to download image from URL")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn().Int("status", resp.StatusCode).Str("url", req.ImageURL).Msg("image download failed")
		Err(c, http.StatusBadGateway, fmt.Sprintf("image download failed with status %d", resp.StatusCode))
		return
	}

	// Validate content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		log.Warn().Str("content_type", contentType).Msg("invalid image content type")
		Err(c, http.StatusBadRequest, "URL does not point to an image (content-type: "+contentType+")")
		return
	}

	// Generate unique filename
	ext := getImageExtension(contentType)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	filepath := filepath.Join(uploadDir, filename)

	// Save the file
	out, err := os.Create(filepath)
	if err != nil {
		log.Error().Err(err).Msg("failed to create file")
		Err(c, http.StatusInternalServerError, "failed to save image")
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		log.Error().Err(err).Msg("failed to write image")
		os.Remove(filepath) // Clean up
		Err(c, http.StatusInternalServerError, "failed to save image")
		return
	}

	// Return local URL (relative path for the frontend to construct full URL)
	localPath := "/uploads/images/" + filename
	log.Info().Str("original_url", req.ImageURL).Str("local_path", localPath).Msg("image uploaded successfully")
	OK(c, gin.H{"url": localPath})
}

// getImageExtension returns the appropriate file extension for a content type
func getImageExtension(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	default:
		return ".jpg"
	}
}

// BulkUploadImages downloads multiple images and returns their local URLs.
//
//	POST /api/v1/articles/upload-images
type BulkImageUploadRequest struct {
	ImageURLs []string `json:"image_urls" binding:"required"`
}

func (h *ArticleHandler) BulkUploadImages(c *gin.Context) {
	var req BulkImageUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Err(c, http.StatusBadRequest, "image_urls is required: "+err.Error())
		return
	}

	if len(req.ImageURLs) > 50 {
		Err(c, http.StatusBadRequest, "maximum 50 images per request")
		return
	}

	// Create uploads directory if it doesn't exist
	uploadDir := "./uploads/images"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Error().Err(err).Msg("failed to create upload directory")
		Err(c, http.StatusInternalServerError, "failed to create upload directory")
		return
	}

	results := make([]map[string]string, 0, len(req.ImageURLs))
	client := &http.Client{Timeout: 30 * time.Second}

	for _, imgURL := range req.ImageURLs {
		imgURL = strings.TrimSpace(imgURL)
		if imgURL == "" {
			continue
		}

		// SSRF protection: reject private/internal URLs
		if isPrivateURL(imgURL) {
			log.Warn().Str("url", imgURL).Msg("SSRF attempt blocked: private URL not allowed")
			results = append(results, map[string]string{"original": imgURL, "local": "", "error": "private URL not allowed"})
			continue
		}

		// Download the image
		resp, err := client.Get(imgURL)
		if err != nil {
			log.Warn().Err(err).Str("url", imgURL).Msg("failed to download image")
			results = append(results, map[string]string{"original": imgURL, "local": "", "error": err.Error()})
			continue
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			resp.Body.Close()
			results = append(results, map[string]string{"original": imgURL, "local": "", "error": "not an image"})
			continue
		}

		ext := getImageExtension(contentType)
		filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
		filepath := filepath.Join(uploadDir, filename)

		out, err := os.Create(filepath)
		if err != nil {
			resp.Body.Close()
			results = append(results, map[string]string{"original": imgURL, "local": "", "error": err.Error()})
			continue
		}

		_, err = io.Copy(out, resp.Body)
		resp.Body.Close()
		out.Close()

		if err != nil {
			os.Remove(filepath)
			results = append(results, map[string]string{"original": imgURL, "local": "", "error": err.Error()})
			continue
		}

		localPath := "/uploads/images/" + filename
		results = append(results, map[string]string{"original": imgURL, "local": localPath})
	}

	OK(c, gin.H{"results": results})
}
