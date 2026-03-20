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
	match := stationPattern.FindStringSubmatch(raw)
	if len(match) != 2 {
		return domain.ClientEvent{}, errors.New("station mac not found")
	}

	eventType := ""
	switch {
	case strings.Contains(raw, " connect "):
		eventType = "connect"
	case strings.Contains(raw, " disconnect "):
		eventType = "disconnect"
	default:
		return domain.ClientEvent{}, errors.New("unsupported event verb")
	}

	return domain.ClientEvent{
		EventDate:  time.Date(receivedAt.Year(), receivedAt.Month(), receivedAt.Day(), 0, 0, 0, 0, receivedAt.Location()),
		EventTime:  receivedAt,
		EventType:  eventType,
		StationMac: strings.ToLower(match[1]),
	}, nil
}
