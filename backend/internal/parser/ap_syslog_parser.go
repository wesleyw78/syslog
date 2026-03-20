package parser

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"syslog/internal/domain"
)

var stationPattern = regexp.MustCompile(`Station\[([^\]]+)\]`)

func ParseAPSyslog(raw string, receivedAt time.Time) (domain.ClientEvent, error) {
	_ = receivedAt

	match := stationPattern.FindStringSubmatch(raw)
	if len(match) != 2 {
		return domain.ClientEvent{}, errors.New("station mac not found")
	}

	eventType := "disconnect"
	if strings.Contains(raw, " connect ") {
		eventType = "connect"
	}

	return domain.ClientEvent{
		EventType:  eventType,
		StationMac: strings.ToLower(match[1]),
	}, nil
}
