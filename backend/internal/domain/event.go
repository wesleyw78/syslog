package domain

import "time"

type SyslogMessage struct {
	ID                uint64
	ReceivedAt        time.Time
	LogTime           *time.Time
	RawMessage        string
	SourceIP          string
	Protocol          string
	ParseStatus       string
	RetentionExpireAt time.Time
}

type ClientEvent struct {
	ID                uint64
	SyslogMessageID   uint64
	EventDate         time.Time
	EventTime         time.Time
	EventType         string
	StationMac        string
	APMac             string
	SSID              string
	IPv4              string
	IPv6              string
	Hostname          string
	OSVendor          string
	MatchedEmployeeID *uint64
	MatchStatus       string
}
