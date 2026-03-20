package config

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	cfg := LoadConfigFromEnv(func(string) string { return "" })
	if cfg.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected default timezone Asia/Shanghai, got %s", cfg.Timezone)
	}
	if cfg.SyslogRetentionDays != 30 {
		t.Fatalf("expected retention 30, got %d", cfg.SyslogRetentionDays)
	}
}
