// Package metrics provides Prometheus metrics collection for the secflow platform.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RecordHTTPRequest records an HTTP request metric.
func RecordHTTPRequest(method, path, status string, duration float64) {
	APIRequestsTotal.WithLabelValues(method, path, status).Inc()
	APIRequestDuration.WithLabelValues(method, path, status).Observe(duration)
}

var (
	// Task queue metrics
	TaskQueueSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "secflow_task_queue_size",
		Help: "Number of tasks in queue by type",
	}, []string{"queue_type"}) // pending, priority, retry

	TaskProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "secflow_task_processing_duration_seconds",
		Help:    "Task processing duration",
		Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800}, // Up to 30 minutes
	}, []string{"task_type", "status"}) // vuln_crawl, article_crawl | done, failed

	TaskRetryCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "secflow_task_retry_total",
		Help: "Total number of task retries",
	}, []string{"task_type", "retry_reason"}) // client_error, timeout, network_error

	TaskCreatedCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "secflow_task_created_total",
		Help: "Total number of tasks created",
	}, []string{"task_type", "priority"}) // vuln_crawl, article_crawl | high, medium, low

	// Node metrics
	NodeTaskCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "secflow_node_task_count",
		Help: "Number of tasks running on node",
	}, []string{"node_id", "node_name"})

	NodeHeartbeat = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "secflow_node_heartbeat_timestamp",
		Help: "Last heartbeat timestamp",
	}, []string{"node_id", "node_name"})

	NodeCPUPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "secflow_node_cpu_percent",
		Help: "Node CPU usage percentage",
	}, []string{"node_id", "node_name"})

	NodeMemoryPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "secflow_node_memory_percent",
		Help: "Node memory usage percentage",
	}, []string{"node_id", "node_name"})

	NodeSuccessRate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "secflow_node_success_rate",
		Help: "Node task success rate (0.0 to 1.0)",
	}, []string{"node_id", "node_name"})

	// Scheduler metrics
	SchedulerDispatchCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "secflow_scheduler_dispatch_total",
		Help: "Total number of task dispatch attempts",
	}, []string{"result"}) // success, no_nodes, no_tasks

	SchedulerBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "secflow_scheduler_batch_size",
		Help:    "Number of tasks dispatched in a batch",
		Buckets: []float64{1, 2, 3, 5, 10, 20, 50},
	})

	// API metrics
	APIRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "secflow_api_request_duration_seconds",
		Help:    "API request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "endpoint", "status"})

	APIRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "secflow_api_requests_total",
		Help: "Total number of API requests",
	}, []string{"method", "endpoint", "status"})
)
