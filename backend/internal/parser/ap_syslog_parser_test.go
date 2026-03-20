package parser

import (
	"testing"
	"time"
)

func TestParseConnectEvent(t *testing.T) {
	raw := "Mar 21 00:33:38 stamgr: Mef85d2S4D0 client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[WesleyHomeEquipment] osvendor[Unknown] hostname[Wesley17PM]"
	event, err := ParseAPSyslog(raw, time.Date(2026, 3, 21, 0, 33, 38, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	if event.EventType != "connect" || event.StationMac != "94:89:78:55:9a:f3" {
		t.Fatalf("unexpected parse result: %#v", event)
	}
}
