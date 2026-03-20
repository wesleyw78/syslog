package scheduler

import (
	"time"

	"syslog/internal/domain"
	"syslog/internal/service"
)

type Cron struct {
	dayEndService *service.DayEndService
}

func NewCron(dayEndService *service.DayEndService) *Cron {
	if dayEndService == nil {
		dayEndService = service.NewDayEndService()
	}

	return &Cron{dayEndService: dayEndService}
}

func (c *Cron) RunDayEnd(now time.Time, records []domain.AttendanceRecord) []domain.AttendanceRecord {
	finalized := make([]domain.AttendanceRecord, 0, len(records))
	for _, record := range records {
		finalized = append(finalized, c.dayEndService.FinalizeForDay(record, now))
	}

	return finalized
}
