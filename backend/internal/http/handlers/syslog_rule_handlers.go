package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
	"syslog/internal/service"
)

type SyslogRuleReader interface {
	List(context.Context) ([]domain.SyslogReceiveRule, error)
}

type SyslogRuleAdminWriter interface {
	CreateRule(context.Context, service.SyslogReceiveRuleWriteInput) (*domain.SyslogReceiveRule, error)
	UpdateRule(context.Context, uint64, service.SyslogReceiveRuleWriteInput) (*domain.SyslogReceiveRule, error)
	DeleteRule(context.Context, uint64) error
	MoveRule(context.Context, uint64, string) (*domain.SyslogReceiveRule, error)
	PreviewRule(context.Context, service.SyslogRulePreviewInput) (*service.SyslogRulePreviewResult, error)
}

type syslogRuleWriteRequest struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	EventType       string `json:"eventType"`
	MessagePattern  string `json:"messagePattern"`
	StationMacGroup string `json:"stationMacGroup"`
	APMacGroup      string `json:"apMacGroup"`
	SSIDGroup       string `json:"ssidGroup"`
	IPv4Group       string `json:"ipv4Group"`
	IPv6Group       string `json:"ipv6Group"`
	HostnameGroup   string `json:"hostnameGroup"`
	OSVendorGroup   string `json:"osVendorGroup"`
	EventTimeGroup  string `json:"eventTimeGroup"`
	EventTimeLayout string `json:"eventTimeLayout"`
}

type syslogRuleMoveRequest struct {
	Direction string `json:"direction"`
}

type syslogRulePreviewRequest struct {
	ReceivedAt string                 `json:"receivedAt"`
	RawMessage string                 `json:"rawMessage"`
	Rule       syslogRuleWriteRequest `json:"rule"`
}

func NewSyslogRulesHandler(repo repository.SyslogReceiveRuleRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if repo == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		items, err := repo.List(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		values := make([]any, 0, len(items))
		for _, item := range items {
			values = append(values, item)
		}
		writeJSON(w, http.StatusOK, listResponse{Items: values})
	}
}

func NewSyslogRuleCreateHandler(admin SyslogRuleAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var req syslogRuleWriteRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rule, err := admin.CreateRule(r.Context(), syslogRuleWriteInput(req))
		if err != nil {
			status := statusCodeForServiceError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}

		writeJSON(w, http.StatusCreated, rule)
	}
}

func NewSyslogRuleUpdateHandler(admin SyslogRuleAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		id, err := parseUint64PathValue(r, "id")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var req syslogRuleWriteRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rule, err := admin.UpdateRule(r.Context(), id, syslogRuleWriteInput(req))
		if err != nil {
			status := statusCodeForServiceError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}

		writeJSON(w, http.StatusOK, rule)
	}
}

func NewSyslogRuleDeleteHandler(admin SyslogRuleAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		id, err := parseUint64PathValue(r, "id")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if err := admin.DeleteRule(r.Context(), id); err != nil {
			status := statusCodeForServiceError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func NewSyslogRuleMoveHandler(admin SyslogRuleAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		id, err := parseUint64PathValue(r, "id")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var req syslogRuleMoveRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rule, err := admin.MoveRule(r.Context(), id, strings.TrimSpace(req.Direction))
		if err != nil {
			status := statusCodeForServiceError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}

		writeJSON(w, http.StatusOK, rule)
	}
}

func NewSyslogRulePreviewHandler(admin SyslogRuleAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var req syslogRulePreviewRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		receivedAt := time.Now()
		if strings.TrimSpace(req.ReceivedAt) != "" {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ReceivedAt))
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			receivedAt = parsed
		}

		result, err := admin.PreviewRule(r.Context(), service.SyslogRulePreviewInput{
			ReceivedAt: receivedAt,
			RawMessage: req.RawMessage,
			Rule:       syslogRuleWriteInput(req.Rule),
		})
		if err != nil {
			status := statusCodeForServiceError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func syslogRuleWriteInput(req syslogRuleWriteRequest) service.SyslogReceiveRuleWriteInput {
	return service.SyslogReceiveRuleWriteInput{
		Name:            req.Name,
		Enabled:         req.Enabled,
		EventType:       req.EventType,
		MessagePattern:  req.MessagePattern,
		StationMacGroup: req.StationMacGroup,
		APMacGroup:      req.APMacGroup,
		SSIDGroup:       req.SSIDGroup,
		IPv4Group:       req.IPv4Group,
		IPv6Group:       req.IPv6Group,
		HostnameGroup:   req.HostnameGroup,
		OSVendorGroup:   req.OSVendorGroup,
		EventTimeGroup:  req.EventTimeGroup,
		EventTimeLayout: req.EventTimeLayout,
	}
}
