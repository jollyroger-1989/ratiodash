package domain

import (
	"context"
	"time"
)

// Report is a scheduled notification that sends a digest of current tracker stats
// to one or more notifier backends.
type Report struct {
	ID              uint             `json:"id"               gorm:"primaryKey"`
	Name            string           `json:"name"`
	CronExpr        string           `json:"cron_expr"`
	LastSentAt      *time.Time       `json:"last_sent_at"`
	NotifierConfigs []NotifierConfig `json:"notifier_configs" gorm:"many2many:report_notifier_configs"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// CreateReportInput holds the fields required to create a new report.
type CreateReportInput struct {
	Name              string `json:"name"               required:"true" minLength:"1" doc:"Human-readable report name"`
	CronExpr          string `json:"cron_expr"          required:"true" minLength:"1" doc:"Cron schedule expression"`
	NotifierConfigIDs []uint `json:"notifier_config_ids"                              doc:"IDs of notifier configs to deliver the report to"`
}

// UpdateReportInput holds the optional fields for patching a report.
type UpdateReportInput struct {
	Name              *string `json:"name"`
	CronExpr          *string `json:"cron_expr"`
	NotifierConfigIDs *[]uint `json:"notifier_config_ids"`
}

// ReportRepository is the persistence abstraction for Report.
type ReportRepository interface {
	FindAll() ([]Report, error)
	FindByID(id uint) (*Report, error)
	Create(report *Report) error
	Update(report *Report) error
	UpdateNotifierConfigs(reportID uint, configIDs []uint) error
	Delete(id uint) error
	UpdateLastSentAt(id uint, sentAt time.Time) error
}

// ReportService is the business-logic abstraction for reports.
type ReportService interface {
	GetAll() ([]Report, error)
	GetByID(id uint) (*Report, error)
	Create(input CreateReportInput) (*Report, error)
	Update(id uint, input UpdateReportInput) (*Report, error)
	Delete(id uint) error
	// Send immediately generates and dispatches a report to all its notifiers.
	Send(ctx context.Context, id uint) error
}

// ReportScheduler manages the live cron schedule of report dispatches.
// Implemented by internal/scheduler.
type ReportScheduler interface {
	ScheduleReport(report Report) error
	UnscheduleReport(reportID uint)
}
