// Package articlegrabber provides article crawlers for security news and articles.
package articlegrabber

import (
	"context"
	"sync"
	"time"
)

// HealthChecker monitors the health of registered grabbers.
type HealthChecker struct {
	grabbers map[string]Grabber
	results  map[string]*HealthStatus
	mu       sync.RWMutex
}

// HealthStatus represents the health status of a single grabber.
type HealthStatus struct {
	Name        string        `json:"name"`
	LastCheck   time.Time     `json:"last_check"`
	IsHealthy   bool          `json:"is_healthy"`
	LastError   string        `json:"last_error,omitempty"`
	AvgLatency  time.Duration `json:"avg_latency"`
	SuccessRate float64       `json:"success_rate"`
	TotalRequests int          `json:"total_requests"`
	TotalFailures int          `json:"total_failures"`
}

// HealthStats provides aggregate health statistics across all grabbers.
type HealthStats struct {
	TotalGrabbers   int     `json:"total_grabbers"`
	HealthyCount   int     `json:"healthy_count"`
	UnhealthyCount int     `json:"unhealthy_count"`
	OverallSuccess float64 `json:"overall_success_rate"`
	AvgLatency     time.Duration `json:"avg_latency"`
}

// NewHealthChecker creates a new HealthChecker for monitoring grabbers.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		grabbers: make(map[string]Grabber),
		results:  make(map[string]*HealthStatus),
	}
}

// Register adds a grabber to health monitoring.
func (h *HealthChecker) Register(name string, grabber Grabber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.grabbers[name] = grabber
	if _, exists := h.results[name]; !exists {
		h.results[name] = &HealthStatus{
			Name:        name,
			TotalRequests: 0,
			TotalFailures: 0,
			SuccessRate:  1.0,
		}
	}
}

// RegisterFromRegistry registers all grabbers from the global registry.
func (h *HealthChecker) RegisterFromRegistry() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, grabber := range registry {
		h.grabbers[name] = grabber
		if _, exists := h.results[name]; !exists {
			h.results[name] = &HealthStatus{
				Name:        name,
				TotalRequests: 0,
				TotalFailures: 0,
				SuccessRate:  1.0,
			}
		}
	}
}

// Check performs a health check on a specific grabber.
// Returns the health status after the check.
func (h *HealthChecker) Check(source string) *HealthStatus {
	h.mu.RLock()
	grabber, exists := h.grabbers[source]
	h.mu.RUnlock()

	if !exists {
		return &HealthStatus{
			Name:      source,
			IsHealthy: false,
			LastError: "grabber not found",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	// Perform a test fetch with limit 1
	_, err := grabber.Fetch(ctx, 1)

	latency := time.Since(start)

	h.mu.Lock()
	defer h.mu.Unlock()

	status := h.results[source]
	if status == nil {
		status = &HealthStatus{Name: source}
		h.results[source] = status
	}

	status.LastCheck = time.Now()
	status.AvgLatency = (status.AvgLatency*time.Duration(status.TotalRequests) + latency) / time.Duration(status.TotalRequests+1)
	status.TotalRequests++

	if err != nil {
		status.IsHealthy = false
		status.LastError = err.Error()
		status.TotalFailures++
	} else {
		status.IsHealthy = true
		status.LastError = ""
	}

	// Update success rate
	if status.TotalRequests > 0 {
		status.SuccessRate = float64(status.TotalRequests-status.TotalFailures) / float64(status.TotalRequests)
	}

	return status
}

// CheckAll performs health checks on all registered grabbers concurrently.
// Returns a map of source names to their health status.
func (h *HealthChecker) CheckAll() map[string]*HealthStatus {
	h.mu.RLock()
	names := make([]string, 0, len(h.grabbers))
	for name := range h.grabbers {
		names = append(names, name)
	}
	h.mu.RUnlock()

	results := make(map[string]*HealthStatus)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			status := h.Check(n)
			mu.Lock()
			results[n] = status
			mu.Unlock()
		}(name)
	}

	wg.Wait()
	return results
}

// IsHealthy returns whether a grabber is currently healthy.
func (h *HealthChecker) IsHealthy(source string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if status, exists := h.results[source]; exists {
		return status.IsHealthy
	}
	return false
}

// GetStatus returns the current health status for a grabber.
func (h *HealthChecker) GetStatus(source string) *HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if status, exists := h.results[source]; exists {
		// Return a copy to avoid race conditions
		copy := *status
		return &copy
	}
	return nil
}

// GetStats returns aggregate health statistics across all grabbers.
func (h *HealthChecker) GetStats() *HealthStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := &HealthStats{
		TotalGrabbers: len(h.grabbers),
	}

	var totalRequests, totalFailures int
	var totalLatency time.Duration

	for _, status := range h.results {
		if status.IsHealthy {
			stats.HealthyCount++
		} else {
			stats.UnhealthyCount++
		}
		totalRequests += status.TotalRequests
		totalFailures += status.TotalFailures
		totalLatency += status.AvgLatency
	}

	if totalRequests > 0 {
		stats.OverallSuccess = float64(totalRequests-totalFailures) / float64(totalRequests)
	}

	if stats.TotalGrabbers > 0 {
		stats.AvgLatency = totalLatency / time.Duration(stats.TotalGrabbers)
	}

	return stats
}

// RecordSuccess records a successful fetch for a grabber.
func (h *HealthChecker) RecordSuccess(source string, latency time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	status := h.results[source]
	if status == nil {
		status = &HealthStatus{Name: source}
		h.results[source] = status
	}

	status.TotalRequests++
	status.AvgLatency = (status.AvgLatency*time.Duration(status.TotalRequests-1) + latency) / time.Duration(status.TotalRequests)
	status.IsHealthy = true
	status.LastError = ""
	status.SuccessRate = float64(status.TotalRequests-status.TotalFailures) / float64(status.TotalRequests)
}

// RecordFailure records a failed fetch for a grabber.
func (h *HealthChecker) RecordFailure(source string, latency time.Duration, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	status := h.results[source]
	if status == nil {
		status = &HealthStatus{Name: source}
		h.results[source] = status
	}

	status.TotalRequests++
	status.TotalFailures++
	status.AvgLatency = (status.AvgLatency*time.Duration(status.TotalRequests-1) + latency) / time.Duration(status.TotalRequests)
	status.IsHealthy = false
	if err != nil {
		status.LastError = err.Error()
	}
	status.SuccessRate = float64(status.TotalRequests-status.TotalFailures) / float64(status.TotalRequests)
}

// Reset clears all health status data.
func (h *HealthChecker) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.results = make(map[string]*HealthStatus)
}

// ResetSource clears health status for a specific grabber.
func (h *HealthChecker) ResetSource(source string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if status, exists := h.results[source]; exists {
		status.TotalRequests = 0
		status.TotalFailures = 0
		status.SuccessRate = 1.0
		status.AvgLatency = 0
		status.IsHealthy = true
		status.LastError = ""
	}
}

// ListSources returns all registered grabber names.
func (h *HealthChecker) ListSources() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	names := make([]string, 0, len(h.grabbers))
	for name := range h.grabbers {
		names = append(names, name)
	}
	return names
}