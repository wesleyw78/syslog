package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"syslog/internal/bootstrap"
	httpapi "syslog/internal/http"
	"syslog/internal/ingest"
	"syslog/internal/scheduler"
	"syslog/internal/service"
)

const adminHTTPAddr = ":8080"

func main() {
	app, err := bootstrap.New(os.Getenv)
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("close app: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dayEndService := service.NewDayEndService()
	dayEndCron := scheduler.NewCron(dayEndService)
	adminServer := httpapi.NewServer(adminHTTPAddr, httpapi.Dependencies{
		Employees:       app.Repositories.Employees,
		EmployeeAdmin:   app.Services.EmployeeAdmin,
		SyslogMessages:  app.Repositories.SyslogMessages,
		ClientEvents:    app.Repositories.ClientEvents,
		Logs:            app.Repositories.Logs,
		Attendance:      app.Repositories.Attendance,
		AttendanceAdmin: app.Services.AttendanceAdmin,
		DebugAdmin:      app.Services.DebugAdmin,
		Settings:        app.Repositories.Settings,
		SettingsAdmin:   app.Services.SettingsAdmin,
		SyslogRules:     app.Repositories.SyslogRules,
		SyslogRuleAdmin: app.Services.SyslogRuleAdmin,
	})
	udpListener := ingest.NewUDPListener(app.Config.SyslogUDPAddr, func(ctx context.Context, payload []byte, addr net.Addr) error {
		receivedAt := time.Now()
		if app.Location != nil {
			receivedAt = receivedAt.In(app.Location)
		}

		return app.Services.SyslogPipeline.Handle(ctx, payload, addr, receivedAt)
	})

	if err := udpListener.Start(); err != nil {
		log.Fatalf("start udp listener: %v", err)
	}
	defer func() {
		if err := udpListener.Close(); err != nil {
			log.Printf("close udp listener: %v", err)
		}
	}()

	go func() {
		if err := udpListener.Serve(ctx); err != nil {
			log.Printf("udp listener stopped: %v", err)
			stop()
		}
	}()

	go func() {
		if err := adminServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("admin http server stopped: %v", err)
			stop()
		}
	}()

	if app.Services.ReportDispatcher != nil {
		go func() {
			if err := app.Services.ReportDispatcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("attendance report dispatcher stopped: %v", err)
				stop()
			}
		}()
	}

	log.Printf(
		"syslog backend bootstrap ready: timezone=%s retention_days=%d scheduler=%T admin_http=%s syslog_udp=%s",
		app.Config.Timezone,
		app.Config.SyslogRetentionDays,
		dayEndCron,
		adminHTTPAddr,
		app.Config.SyslogUDPAddr,
	)
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := adminServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("shutdown admin http server: %v", err)
	}
}
