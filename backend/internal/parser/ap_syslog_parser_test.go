package parser

import (
	"testing"
	"time"
)

func TestParseConnectEvent(t *testing.T) {
	receivedAt := time.Date(2026, 3, 21, 0, 33, 38, 0, time.FixedZone("CST", 8*3600))
	raw := "Mar 21 00:33:38 stamgr: Mef85d2S4D0 client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[WesleyHomeEquipment] osvendor[Unknown] hostname[Wesley17PM]"

	event, err := ParseAPSyslog(raw, receivedAt)
	if err != nil {
		t.Fatal(err)
	}

	expectedEventDate := time.Date(2026, 3, 21, 0, 0, 0, 0, receivedAt.Location())
	if event.EventType != "connect" || event.StationMac != "94:89:78:55:9a:f3" || !event.EventDate.Equal(expectedEventDate) || !event.EventTime.Equal(receivedAt) {
		t.Fatalf("unexpected parse result: %#v", event)
	}
}

func TestParseDisconnectEvent(t *testing.T) {
	receivedAt := time.Date(2026, 3, 21, 1, 2, 3, 0, time.FixedZone("CST", 8*3600))
	raw := "Mar 21 01:02:03 stamgr: Mef85d2S4D0 client_footprints disconnect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[WesleyHomeEquipment] osvendor[Unknown] hostname[Wesley17PM]"

	event, err := ParseAPSyslog(raw, receivedAt)
	if err != nil {
		t.Fatal(err)
	}

	if event.EventType != "disconnect" || event.StationMac != "94:89:78:55:9a:f3" {
		t.Fatalf("unexpected parse result: %#v", event)
	}
}

func TestParseAPSyslogMissingStationReturnsError(t *testing.T) {
	raw := "Mar 21 00:33:38 stamgr: Mef85d2S4D0 client_footprints connect AP[28:b3:71:25:ae:a0] ssid[WesleyHomeEquipment]"

	_, err := ParseAPSyslog(raw, time.Date(2026, 3, 21, 0, 33, 38, 0, time.FixedZone("CST", 8*3600)))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseAPSyslogUnsupportedVerbReturnsError(t *testing.T) {
	raw := "Mar 21 00:33:38 stamgr: Mef85d2S4D0 client_footprints roam Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[WesleyHomeEquipment] osvendor[Unknown] hostname[Wesley17PM]"

	_, err := ParseAPSyslog(raw, time.Date(2026, 3, 21, 0, 33, 38, 0, time.FixedZone("CST", 8*3600)))
	if err == nil {
		t.Fatal("expected error")
	}
}
