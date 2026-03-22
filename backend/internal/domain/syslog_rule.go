package domain

import "time"

type SyslogReceiveRule struct {
	ID              uint64    `json:"id"`
	SortOrder       uint32    `json:"sortOrder"`
	Name            string    `json:"name"`
	Enabled         bool      `json:"enabled"`
	EventType       string    `json:"eventType"`
	MessagePattern  string    `json:"messagePattern"`
	StationMacGroup string    `json:"stationMacGroup"`
	APMacGroup      string    `json:"apMacGroup"`
	SSIDGroup       string    `json:"ssidGroup"`
	IPv4Group       string    `json:"ipv4Group"`
	IPv6Group       string    `json:"ipv6Group"`
	HostnameGroup   string    `json:"hostnameGroup"`
	OSVendorGroup   string    `json:"osVendorGroup"`
	EventTimeGroup  string    `json:"eventTimeGroup"`
	EventTimeLayout string    `json:"eventTimeLayout"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
