package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

var ErrInvalidSyslogRuleInput = errors.New("invalid syslog rule input")

type SyslogReceiveRuleWriteInput struct {
	Name            string
	Enabled         bool
	EventType       string
	MessagePattern  string
	StationMacGroup string
	APMacGroup      string
	SSIDGroup       string
	IPv4Group       string
	IPv6Group       string
	HostnameGroup   string
	OSVendorGroup   string
	EventTimeGroup  string
	EventTimeLayout string
}

type SyslogRulePreviewInput struct {
	ReceivedAt time.Time
	RawMessage string
	Rule       SyslogReceiveRuleWriteInput
}

type SyslogRulePreviewResult struct {
	Matched bool                `json:"matched"`
	Event   *domain.ClientEvent `json:"event,omitempty"`
}

type SyslogRuleAdminService struct {
	repo repository.SyslogReceiveRuleRepository
}

func NewSyslogRuleAdminService(repo repository.SyslogReceiveRuleRepository) *SyslogRuleAdminService {
	return &SyslogRuleAdminService{repo: repo}
}

func (s *SyslogRuleAdminService) CreateRule(ctx context.Context, input SyslogReceiveRuleWriteInput) (*domain.SyslogReceiveRule, error) {
	if s.repo == nil {
		return nil, errors.New("syslog rule repository is required")
	}

	rule, err := normalizeSyslogRuleInput(0, input)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *SyslogRuleAdminService) UpdateRule(ctx context.Context, id uint64, input SyslogReceiveRuleWriteInput) (*domain.SyslogReceiveRule, error) {
	if s.repo == nil {
		return nil, errors.New("syslog rule repository is required")
	}
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return nil, err
	}

	rule, err := normalizeSyslogRuleInput(id, input)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *SyslogRuleAdminService) DeleteRule(ctx context.Context, id uint64) error {
	if s.repo == nil {
		return errors.New("syslog rule repository is required")
	}
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *SyslogRuleAdminService) MoveRule(ctx context.Context, id uint64, direction string) (*domain.SyslogReceiveRule, error) {
	if s.repo == nil {
		return nil, errors.New("syslog rule repository is required")
	}
	if direction != "up" && direction != "down" {
		return nil, fmt.Errorf("%w: direction must be up or down", ErrInvalidSyslogRuleInput)
	}
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return nil, err
	}
	if err := s.repo.Move(ctx, id, direction); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *SyslogRuleAdminService) PreviewRule(_ context.Context, input SyslogRulePreviewInput) (*SyslogRulePreviewResult, error) {
	rule, err := normalizeSyslogRuleInput(0, input.Rule)
	if err != nil {
		return nil, err
	}

	event, matched, err := applySyslogRule(strings.TrimSpace(input.RawMessage), input.ReceivedAt, rule)
	if err != nil {
		return nil, err
	}

	return &SyslogRulePreviewResult{
		Matched: matched,
		Event:   event,
	}, nil
}

func normalizeSyslogRuleInput(id uint64, input SyslogReceiveRuleWriteInput) (domain.SyslogReceiveRule, error) {
	rule := domain.SyslogReceiveRule{
		ID:              id,
		Name:            strings.TrimSpace(input.Name),
		Enabled:         input.Enabled,
		EventType:       strings.TrimSpace(input.EventType),
		MessagePattern:  strings.TrimSpace(input.MessagePattern),
		StationMacGroup: strings.TrimSpace(input.StationMacGroup),
		APMacGroup:      strings.TrimSpace(input.APMacGroup),
		SSIDGroup:       strings.TrimSpace(input.SSIDGroup),
		IPv4Group:       strings.TrimSpace(input.IPv4Group),
		IPv6Group:       strings.TrimSpace(input.IPv6Group),
		HostnameGroup:   strings.TrimSpace(input.HostnameGroup),
		OSVendorGroup:   strings.TrimSpace(input.OSVendorGroup),
		EventTimeGroup:  strings.TrimSpace(input.EventTimeGroup),
		EventTimeLayout: strings.TrimSpace(input.EventTimeLayout),
	}

	if rule.Name == "" {
		return domain.SyslogReceiveRule{}, fmt.Errorf("%w: name is required", ErrInvalidSyslogRuleInput)
	}
	if rule.EventType != "connect" && rule.EventType != "disconnect" {
		return domain.SyslogReceiveRule{}, fmt.Errorf("%w: eventType must be connect or disconnect", ErrInvalidSyslogRuleInput)
	}
	if rule.MessagePattern == "" {
		return domain.SyslogReceiveRule{}, fmt.Errorf("%w: messagePattern is required", ErrInvalidSyslogRuleInput)
	}
	if rule.StationMacGroup == "" {
		return domain.SyslogReceiveRule{}, fmt.Errorf("%w: stationMacGroup is required", ErrInvalidSyslogRuleInput)
	}
	if rule.EventTimeGroup == "" {
		rule.EventTimeLayout = ""
	}
	if rule.EventTimeGroup != "" && rule.EventTimeLayout == "" {
		return domain.SyslogReceiveRule{}, fmt.Errorf("%w: eventTimeLayout is required when eventTimeGroup is set", ErrInvalidSyslogRuleInput)
	}

	compiled, err := regexp.Compile(rule.MessagePattern)
	if err != nil {
		return domain.SyslogReceiveRule{}, fmt.Errorf("%w: invalid messagePattern: %v", ErrInvalidSyslogRuleInput, err)
	}

	availableGroups := make(map[string]struct{})
	for _, name := range compiled.SubexpNames() {
		if name == "" {
			continue
		}
		availableGroups[name] = struct{}{}
	}

	for field, group := range map[string]string{
		"stationMacGroup": rule.StationMacGroup,
		"apMacGroup":      rule.APMacGroup,
		"ssidGroup":       rule.SSIDGroup,
		"ipv4Group":       rule.IPv4Group,
		"ipv6Group":       rule.IPv6Group,
		"hostnameGroup":   rule.HostnameGroup,
		"osVendorGroup":   rule.OSVendorGroup,
		"eventTimeGroup":  rule.EventTimeGroup,
	} {
		if group == "" {
			continue
		}
		if _, ok := availableGroups[group]; !ok {
			return domain.SyslogReceiveRule{}, fmt.Errorf("%w: %s references missing regex group %s", ErrInvalidSyslogRuleInput, field, group)
		}
	}

	return rule, nil
}
