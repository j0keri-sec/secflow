package report

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/secflow/server/internal/model"
)

// Scheduler handles automatic report generation.
type Scheduler struct {
	gen          *Generator
	reportRepo   ReportRepoInterface
	stopCh       chan struct{}
	wg           sync.WaitGroup
	config       *SchedulerConfig
}

// SchedulerConfig holds configuration for the report scheduler.
type SchedulerConfig struct {
	// Weekly report generation
	WeeklyEnabled bool
	WeeklyCron    string // cron expression, e.g., "0 9 * * 1" for Monday 9am

	// Monthly report generation
	MonthlyEnabled bool
	MonthlyCron    string // cron expression, e.g., "0 9 1 * *" for 1st of month

	// Data sources to include
	DefaultSources []DataSource

	// AI model to use
	AIModel AIModelType

	// Notification callback
	OnReportGenerated func(reportID string, err error)
}

// ReportRepoInterface defines the interface for report storage.
type ReportRepoInterface interface {
	Create(ctx context.Context, report *model.Report) error
}

// NewScheduler creates a new report scheduler.
func NewScheduler(gen *Generator, reportRepo ReportRepoInterface, config *SchedulerConfig) *Scheduler {
	return &Scheduler{
		gen:        gen,
		reportRepo: reportRepo,
		config:     config,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the scheduled report generation.
func (s *Scheduler) Start() {
	if s.config.WeeklyEnabled {
		s.wg.Add(1)
		go s.runWeeklyScheduler()
	}

	if s.config.MonthlyEnabled {
		s.wg.Add(1)
		go s.runMonthlyScheduler()
	}
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// runWeeklyScheduler runs the weekly report generation loop.
func (s *Scheduler) runWeeklyScheduler() {
	defer s.wg.Done()

	// Calculate next Monday 9am
	now := time.Now()
	next := getNextWeeklyRun(now, time.Monday, 9, 0)
	
	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			s.generateWeeklyReport()
			// Schedule next week
			next = getNextWeeklyRun(time.Now(), time.Monday, 9, 0)
			timer.Reset(time.Until(next))
		}
	}
}

// runMonthlyScheduler runs the monthly report generation loop.
func (s *Scheduler) runMonthlyScheduler() {
	defer s.wg.Done()

	// Calculate next 1st of month 9am
	now := time.Now()
	next := getNextMonthlyRun(now, 1, 9, 0)

	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			s.generateMonthlyReport()
			// Schedule next month
			next = getNextMonthlyRun(time.Now(), 1, 9, 0)
			timer.Reset(time.Until(next))
		}
	}
}

// generateWeeklyReport generates the weekly report.
func (s *Scheduler) generateWeeklyReport() {
	ctx := context.Background()
	
	// Calculate date range (last 7 days)
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -7)

	// Generate report data
	data, err := s.gen.GenerateReport(ctx, &ReportConfig{
		Title:      fmt.Sprintf("第%d期安全周报", getWeekNumber(dateTo)),
		ReportType: ReportTypeWeekly,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Sources:    s.config.DefaultSources,
		AIModel:   s.config.AIModel,
	})

	if err != nil {
		if s.config.OnReportGenerated != nil {
			s.config.OnReportGenerated("", err)
		}
		return
	}

	// Generate HTML content
	content, _, err := s.gen.ExportToFile(data, FormatHTML)
	if err != nil {
		if s.config.OnReportGenerated != nil {
			s.config.OnReportGenerated("", err)
		}
		return
	}

	// Create report record
	report := &model.Report{
		Title:    data.Title,
		Period:   fmt.Sprintf("%s ~ %s", dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02")),
		Content:  string(content),
		Status:   model.ReportDone,
	}

	if err := s.reportRepo.Create(ctx, report); err != nil {
		if s.config.OnReportGenerated != nil {
			s.config.OnReportGenerated("", err)
		}
		return
	}

	if s.config.OnReportGenerated != nil {
		s.config.OnReportGenerated(report.ID.Hex(), nil)
	}
}

// generateMonthlyReport generates the monthly report.
func (s *Scheduler) generateMonthlyReport() {
	ctx := context.Background()

	// Calculate date range (last month)
	now := time.Now()
	dateTo := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	dateFrom := dateTo.AddDate(0, -1, 0)

	// Generate report data
	data, err := s.gen.GenerateReport(ctx, &ReportConfig{
		Title:      fmt.Sprintf("%d年%d月安全月报", dateFrom.Year(), dateFrom.Month()),
		ReportType: ReportTypeMonthly,
		DateFrom:   dateFrom,
		DateTo:     dateTo.AddDate(0, 0, -1),
		Sources:    s.config.DefaultSources,
		AIModel:   s.config.AIModel,
	})

	if err != nil {
		if s.config.OnReportGenerated != nil {
			s.config.OnReportGenerated("", err)
		}
		return
	}

	// Generate HTML content
	content, _, err := s.gen.ExportToFile(data, FormatHTML)
	if err != nil {
		if s.config.OnReportGenerated != nil {
			s.config.OnReportGenerated("", err)
		}
		return
	}

	// Create report record
	report := &model.Report{
		Title:    data.Title,
		Period:   fmt.Sprintf("%s ~ %s", dateFrom.Format("2006-01-02"), dateTo.AddDate(0, 0, -1).Format("2006-01-02")),
		Content:  string(content),
		Status:   model.ReportDone,
	}

	if err := s.reportRepo.Create(ctx, report); err != nil {
		if s.config.OnReportGenerated != nil {
			s.config.OnReportGenerated("", err)
		}
		return
	}

	if s.config.OnReportGenerated != nil {
		s.config.OnReportGenerated(report.ID.Hex(), nil)
	}
}

// getNextWeeklyRun calculates the next run time for weekly reports.
func getNextWeeklyRun(now time.Time, weekday time.Weekday, hour, minute int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	
	daysUntil := (int(weekday) - int(now.Weekday()) + 7) % 7
	if daysUntil == 0 && now.After(next) {
		daysUntil = 7
	}
	
	return next.AddDate(0, 0, daysUntil)
}

// getNextMonthlyRun calculates the next run time for monthly reports.
func getNextMonthlyRun(now time.Time, day, hour, minute int) time.Time {
	next := time.Date(now.Year(), now.Month(), day, hour, minute, 0, 0, now.Location())
	
	if !now.Before(next) {
		// Already passed this month's date, schedule for next month
		next = next.AddDate(0, 1, 0)
	}
	
	return next
}

// getWeekNumber returns the ISO week number.
func getWeekNumber(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}
