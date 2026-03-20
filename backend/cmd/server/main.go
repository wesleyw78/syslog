package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"syslog/internal/bootstrap"
	"syslog/internal/ingest"
	"syslog/internal/scheduler"
	"syslog/internal/service"
)

func main() {
	app := bootstrap.New(os.Getenv)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dayEndService := service.NewDayEndService()
	dayEndCron := scheduler.NewCron(dayEndService)
	udpListener := ingest.NewUDPListener("", func(ctx context.Context, payload []byte, addr net.Addr) error {
		_ = ctx
		_ = payload
		_ = addr
		return nil
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

	log.Printf(
		"syslog backend bootstrap ready: timezone=%s retention_days=%d scheduler=%T",
		app.Config.Timezone,
		app.Config.SyslogRetentionDays,
		dayEndCron,
	)
	<-ctx.Done()
}
