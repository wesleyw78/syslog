package bootstrap

import (
	"context"
	"database/sql"
	"os"
	"time"

	"syslog/internal/config"
	"syslog/internal/repository"
	"syslog/internal/service"
)

type Repositories struct {
	Employees      repository.EmployeeRepository
	SyslogMessages repository.SyslogMessageRepository
	ClientEvents   repository.ClientEventRepository
	Attendance     repository.AttendanceRepository
	Reports        repository.ReportRepository
	Settings       repository.SystemSettingRepository
}

type Services struct {
	SyslogPipeline  *service.SyslogPipeline
	EmployeeAdmin   *service.EmployeeAdminService
	SettingsAdmin   *service.SettingsAdminService
	AttendanceAdmin *service.AttendanceAdminService
}

type App struct {
	Config       config.Config
	Location     *time.Location
	DB           *sql.DB
	Repositories Repositories
	Services     Services
}

func New(getenv func(string) string) (App, error) {
	if getenv == nil {
		getenv = os.Getenv
	}

	cfg := config.LoadConfigFromEnv(getenv)
	db, err := OpenMySQL(cfg)
	if err != nil {
		return App{}, err
	}

	if err := RunMigrations(context.Background(), db); err != nil {
		_ = db.Close()
		return App{}, err
	}

	loc := mustLoadLocation(cfg.Timezone)
	app := App{
		Config:   cfg,
		Location: loc,
		DB:       db,
		Repositories: Repositories{
			Employees:      repository.NewMySQLEmployeeRepository(db),
			SyslogMessages: repository.NewMySQLSyslogMessageRepository(db),
			ClientEvents:   repository.NewMySQLClientEventRepository(db),
			Attendance:     repository.NewMySQLAttendanceRepository(db),
			Reports:        repository.NewMySQLReportRepository(db),
			Settings:       repository.NewMySQLSystemSettingRepository(db),
		},
	}
	app.Services.EmployeeAdmin = service.NewEmployeeAdminService(db, app.Repositories.Employees)
	app.Services.SettingsAdmin = service.NewSettingsAdminService(db, app.Repositories.Settings)
	app.Services.AttendanceAdmin = service.NewAttendanceAdminService(db, app.Repositories.Attendance, app.Repositories.Reports, service.NewReportService())
	app.Services.SyslogPipeline = service.NewSyslogPipeline(service.SyslogPipelineDeps{
		DB:            db,
		Messages:      app.Repositories.SyslogMessages,
		Events:        app.Repositories.ClientEvents,
		Employees:     app.Repositories.Employees,
		Attendance:    app.Repositories.Attendance,
		Reports:       app.Repositories.Reports,
		Settings:      app.Repositories.Settings,
		RetentionDays: cfg.SyslogRetentionDays,
	})

	return app, nil
}

func (a App) Close() error {
	if a.DB == nil {
		return nil
	}

	return a.DB.Close()
}
