package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"syslog/internal/domain"
)

type SyslogRuleMatchResult struct {
	Rule  *domain.SyslogReceiveRule
	Event *domain.ClientEvent
}

func matchSyslogRule(raw string, receivedAt time.Time, rules []domain.SyslogReceiveRule) (*SyslogRuleMatchResult, error) {
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		event, matched, err := applySyslogRule(raw, receivedAt, rule)
		if err != nil {
			return nil, err
		}
		if matched {
			copiedRule := rule
			return &SyslogRuleMatchResult{
				Rule:  &copiedRule,
				Event: event,
			}, nil
		}
	}

	return nil, nil
}

func applySyslogRule(raw string, receivedAt time.Time, rule domain.SyslogReceiveRule) (*domain.ClientEvent, bool, error) {
	compiled, err := regexp.Compile(rule.MessagePattern)
	if err != nil {
		return nil, false, fmt.Errorf("compile syslog rule %d: %w", rule.ID, err)
	}

	matches := compiled.FindStringSubmatch(raw)
	if len(matches) == 0 {
		return nil, false, nil
	}

	groups := make(map[string]string)
	for idx, name := range compiled.SubexpNames() {
		if idx == 0 || name == "" {
			continue
		}
		groups[name] = strings.TrimSpace(matches[idx])
	}

	stationMac := normalizeMACAddress(groups[rule.StationMacGroup])
	if stationMac == "" {
		return nil, false, fmt.Errorf("syslog rule %d matched but station mac is empty", rule.ID)
	}

	eventTime, err := resolveRuleEventTime(receivedAt, groups, rule)
	if err != nil {
		return nil, false, fmt.Errorf("syslog rule %d event time: %w", rule.ID, err)
	}

	eventDate := time.Date(eventTime.Year(), eventTime.Month(), eventTime.Day(), 0, 0, 0, 0, eventTime.Location())
	return &domain.ClientEvent{
		EventDate:  eventDate,
		EventTime:  eventTime,
		EventType:  rule.EventType,
		StationMac: stationMac,
		APMac:      normalizeMACAddress(groups[rule.APMacGroup]),
		SSID:       groups[rule.SSIDGroup],
		IPv4:       groups[rule.IPv4Group],
		IPv6:       groups[rule.IPv6Group],
		Hostname:   groups[rule.HostnameGroup],
		OSVendor:   groups[rule.OSVendorGroup],
	}, true, nil
}

func resolveRuleEventTime(receivedAt time.Time, groups map[string]string, rule domain.SyslogReceiveRule) (time.Time, error) {
	if rule.EventTimeGroup == "" {
		return receivedAt, nil
	}

	rawEventTime := groups[rule.EventTimeGroup]
	if rawEventTime == "" {
		return time.Time{}, fmt.Errorf("eventTimeGroup %q did not capture a value", rule.EventTimeGroup)
	}

	parsed, err := time.ParseInLocation(rule.EventTimeLayout, rawEventTime, receivedAt.Location())
	if err != nil {
		return time.Time{}, err
	}

	return parsed, nil
}
