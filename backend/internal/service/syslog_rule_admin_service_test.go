package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"syslog/internal/domain"
)

type fakeSyslogRuleRepo struct {
	rules         []domain.SyslogReceiveRule
	created       *domain.SyslogReceiveRule
	updated       *domain.SyslogReceiveRule
	deletedID     uint64
	moveID        uint64
	moveDirection string
	createErr     error
	updateErr     error
	deleteErr     error
	findErr       error
	nextInsertedID uint64
}

func (f *fakeSyslogRuleRepo) List(context.Context) ([]domain.SyslogReceiveRule, error) {
	return append([]domain.SyslogReceiveRule(nil), f.rules...), nil
}

func (f *fakeSyslogRuleRepo) ListEnabled(context.Context) ([]domain.SyslogReceiveRule, error) {
	enabled := make([]domain.SyslogReceiveRule, 0, len(f.rules))
	for _, rule := range f.rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled, nil
}

func (f *fakeSyslogRuleRepo) FindByID(_ context.Context, id uint64) (*domain.SyslogReceiveRule, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	for _, rule := range f.rules {
		if rule.ID == id {
			copied := rule
			return &copied, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (f *fakeSyslogRuleRepo) Create(_ context.Context, rule *domain.SyslogReceiveRule) error {
	if f.createErr != nil {
		return f.createErr
	}
	copied := *rule
	f.created = &copied
	rule.ID = f.nextInsertedID
	if rule.ID == 0 {
		rule.ID = 1
	}
	return nil
}

func (f *fakeSyslogRuleRepo) Update(_ context.Context, rule *domain.SyslogReceiveRule) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	copied := *rule
	f.updated = &copied
	return nil
}

func (f *fakeSyslogRuleRepo) Delete(_ context.Context, id uint64) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedID = id
	return nil
}

func (f *fakeSyslogRuleRepo) Move(_ context.Context, id uint64, direction string) error {
	f.moveID = id
	f.moveDirection = direction
	return nil
}

func TestSyslogRuleAdminServiceCreateRuleNormalizesAndValidatesInput(t *testing.T) {
	repo := &fakeSyslogRuleRepo{nextInsertedID: 7}
	service := NewSyslogRuleAdminService(repo)

	rule, err := service.CreateRule(context.Background(), SyslogReceiveRuleWriteInput{
		Name:            "  AP Connect  ",
		Enabled:         true,
		EventType:       "connect",
		MessagePattern:  `connect Station\[(?P<station_mac>[^\]]+)\] AP\[(?P<ap_mac>[^\]]+)\]`,
		StationMacGroup: "station_mac",
		APMacGroup:      "ap_mac",
	})
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	if rule.ID != 7 {
		t.Fatalf("expected inserted id 7, got %d", rule.ID)
	}
	if repo.created == nil || repo.created.Name != "AP Connect" {
		t.Fatalf("expected trimmed name to be persisted, got %+v", repo.created)
	}
	if repo.created.StationMacGroup != "station_mac" {
		t.Fatalf("expected station mac group to be preserved, got %+v", repo.created)
	}
}

func TestSyslogRuleAdminServiceRejectsUnknownRegexGroup(t *testing.T) {
	service := NewSyslogRuleAdminService(&fakeSyslogRuleRepo{})

	_, err := service.CreateRule(context.Background(), SyslogReceiveRuleWriteInput{
		Name:            "broken",
		Enabled:         true,
		EventType:       "connect",
		MessagePattern:  `connect Station\[(?P<station_mac>[^\]]+)\]`,
		StationMacGroup: "station_mac",
		HostnameGroup:   "hostname",
	})
	if err == nil {
		t.Fatal("expected invalid regex group to be rejected")
	}
}

func TestSyslogRuleAdminServiceDeleteRequiresExistingRule(t *testing.T) {
	repo := &fakeSyslogRuleRepo{}
	service := NewSyslogRuleAdminService(repo)

	if err := service.DeleteRule(context.Background(), 99); err == nil {
		t.Fatal("expected missing rule delete to fail")
	}
}

func TestSyslogRuleAdminServiceMoveRuleRequiresKnownDirection(t *testing.T) {
	repo := &fakeSyslogRuleRepo{
		rules: []domain.SyslogReceiveRule{
			{ID: 11, Name: "connect", Enabled: true, EventType: "connect", MessagePattern: `connect Station\[(?P<station_mac>[^\]]+)\]`, StationMacGroup: "station_mac"},
		},
	}
	service := NewSyslogRuleAdminService(repo)

	if _, err := service.MoveRule(context.Background(), 11, "sideways"); err == nil {
		t.Fatal("expected invalid move direction to be rejected")
	}
}

func TestSyslogRuleAdminServicePreviewRuleReturnsMatchedFields(t *testing.T) {
	service := NewSyslogRuleAdminService(&fakeSyslogRuleRepo{})

	preview, err := service.PreviewRule(context.Background(), SyslogRulePreviewInput{
		ReceivedAt: mustRuleTime(t, "2026-03-22T09:15:00+08:00"),
		RawMessage: "Mar 22 09:15:00 stamgr: client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] hostname[scanner-01]",
		Rule: SyslogReceiveRuleWriteInput{
			Name:            "connect",
			Enabled:         true,
			EventType:       "connect",
			MessagePattern:  `connect Station\[(?P<station_mac>[^\]]+)\] AP\[(?P<ap_mac>[^\]]+)\] ssid\[(?P<ssid>[^\]]+)\] hostname\[(?P<hostname>[^\]]+)\]`,
			StationMacGroup: "station_mac",
			APMacGroup:      "ap_mac",
			SSIDGroup:       "ssid",
			HostnameGroup:   "hostname",
		},
	})
	if err != nil {
		t.Fatalf("expected preview to succeed, got %v", err)
	}
	if !preview.Matched {
		t.Fatal("expected preview to match")
	}
	if preview.Event == nil || preview.Event.EventType != "connect" || preview.Event.StationMac != "94:89:78:55:9a:f3" {
		t.Fatalf("unexpected preview event: %+v", preview.Event)
	}
}

func mustRuleTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
