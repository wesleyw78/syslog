package domain

import "time"

type SyslogMessage struct {
	ID                uint64     `json:"id"`
	ReceivedAt        time.Time  `json:"receivedAt"`
	LogTime           *time.Time `json:"logTime,omitempty"`
	RawMessage        string     `json:"rawMessage"`
	SourceIP          string     `json:"sourceIp"`
	Protocol          string     `json:"protocol"`
	ParseStatus       string     `json:"parseStatus"`
	RetentionExpireAt time.Time  `json:"retentionExpireAt"`
}

type ClientEvent struct {
	ID                uint64    `json:"id"`
	SyslogMessageID   uint64    `json:"syslogMessageId"`
	EventDate         time.Time `json:"eventDate"`
	EventTime         time.Time `json:"eventTime"`
	EventType         string    `json:"eventType"`
	StationMac        string    `json:"stationMac"`
	APMac             string    `json:"apMac"`
	SSID              string    `json:"ssid"`
	IPv4              string    `json:"ipv4"`
	IPv6              string    `json:"ipv6"`
	Hostname          string    `json:"hostname"`
	OSVendor          string    `json:"osVendor"`
	MatchedEmployeeID *uint64   `json:"matchedEmployeeId,omitempty"`
	MatchStatus       string    `json:"matchStatus"`
}
