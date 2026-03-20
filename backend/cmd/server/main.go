package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"syslog/internal/ingest"
	"syslog/internal/scheduler"
	"syslog/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	udpAddr := envOrDefault("UDP_LISTEN_ADDR", ":1514")
	dayEndService := service.NewDayEndService()
	dayEndCron := scheduler.NewCron(dayEndService)
	udpListener := ingest.NewUDPListener(udpAddr, func(ctx context.Context, payload []byte, addr net.Addr) error {
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

	log.Printf("syslog backend bootstrap ready: udp=%s, day-end scheduler=%T", udpAddr, dayEndCron)
	<-ctx.Done()
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
