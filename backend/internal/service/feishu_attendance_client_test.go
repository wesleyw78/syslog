package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSanitizeFeishuHeadersRedactsAuthorization(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer tenant-access-token-value",
		"X-Test":        "visible",
	}

	sanitized := sanitizeFeishuHeaders(headers)

	if strings.Contains(sanitized, "tenant-access-token-value") {
		t.Fatalf("expected authorization token to be redacted, got %q", sanitized)
	}
	if !strings.Contains(sanitized, "Bearer ***") {
		t.Fatalf("expected redacted authorization marker, got %q", sanitized)
	}
	if !strings.Contains(sanitized, "\"X-Test\":\"visible\"") {
		t.Fatalf("expected non-secret headers to remain visible, got %q", sanitized)
	}
}

func TestSanitizeFeishuBodyRedactsAppSecret(t *testing.T) {
	body := map[string]any{
		"app_id":     "cli_123",
		"app_secret": "super-secret-value",
	}

	sanitized := sanitizeFeishuBody(body)

	if strings.Contains(sanitized, "super-secret-value") {
		t.Fatalf("expected app secret to be redacted, got %q", sanitized)
	}
	if !strings.Contains(sanitized, "\"app_secret\":\"***\"") {
		t.Fatalf("expected redacted app secret marker, got %q", sanitized)
	}
	if !strings.Contains(sanitized, "\"app_id\":\"cli_123\"") {
		t.Fatalf("expected app id to remain visible, got %q", sanitized)
	}
}

func TestSanitizeFeishuResponseBodyRedactsTenantToken(t *testing.T) {
	raw := `{"code":0,"tenant_access_token":"tenant-token-value","expire":7200}`

	sanitized := sanitizeFeishuResponseBody(raw)

	if strings.Contains(sanitized, "tenant-token-value") {
		t.Fatalf("expected tenant access token to be redacted, got %q", sanitized)
	}
	if !strings.Contains(sanitized, `"tenant_access_token":"***"`) {
		t.Fatalf("expected redacted tenant access token marker, got %q", sanitized)
	}
}

func TestCreateFlowUsesMinimalFeishuRequestBody(t *testing.T) {
	t.Helper()

	requestBodies := make([]map[string]any, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBodies = append(requestBodies, payload)

		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","tenant_access_token":"token-123","expire":7200}`))
		case "/open-apis/attendance/v1/user_flows/batch_create":
			if got := r.URL.Query().Get("employee_type"); got != "employee_id" {
				t.Fatalf("expected employee_type=employee_id, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"flow_records":[{"record_id":"flow-001"}]}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewFeishuAttendanceHTTPClient(server.Client())
	client.baseURL = server.URL

	_, err := client.CreateFlow(context.Background(), FeishuAttendanceConfig{
		AppID:        "cli_123",
		AppSecret:    "secret_456",
		LocationName: "办公室",
		Timeout:      5 * time.Second,
	}, FeishuAttendanceCreateInput{
		EmployeeType: "employee_id",
		UserID:       "user-001",
		CreatorID:    "user-001",
		LocationName: "办公室",
		CheckTime:    "1774065317",
		Comment:      "Syslog 导入上班打卡",
		ExternalID:   "should-not-be-sent",
		IdempotentID: "should-not-be-sent",
		Type:         7,
	})
	if err != nil {
		t.Fatalf("expected create flow to succeed, got %v", err)
	}

	if len(requestBodies) != 2 {
		t.Fatalf("expected token request and create request, got %d requests", len(requestBodies))
	}

	createPayload := requestBodies[1]
	flowRecords, ok := createPayload["flow_records"].([]any)
	if !ok || len(flowRecords) != 1 {
		t.Fatalf("expected single flow record payload, got %#v", createPayload["flow_records"])
	}

	record, ok := flowRecords[0].(map[string]any)
	if !ok {
		t.Fatalf("expected flow record object, got %#v", flowRecords[0])
	}

	if len(record) != 5 {
		t.Fatalf("expected minimal 5-field request body, got %#v", record)
	}
	if record["user_id"] != "user-001" {
		t.Fatalf("expected user_id user-001, got %#v", record["user_id"])
	}
	if record["creator_id"] != "user-001" {
		t.Fatalf("expected creator_id user-001, got %#v", record["creator_id"])
	}
	if record["location_name"] != "办公室" {
		t.Fatalf("expected location_name 办公室, got %#v", record["location_name"])
	}
	if record["check_time"] != "1774065317" {
		t.Fatalf("expected check_time 1774065317, got %#v", record["check_time"])
	}
	if record["comment"] != "Syslog 导入上班打卡" {
		t.Fatalf("expected comment to propagate, got %#v", record["comment"])
	}
	if _, exists := record["external_id"]; exists {
		t.Fatalf("expected external_id to be omitted, got %#v", record["external_id"])
	}
	if _, exists := record["idempotent_id"]; exists {
		t.Fatalf("expected idempotent_id to be omitted, got %#v", record["idempotent_id"])
	}
	if _, exists := record["type"]; exists {
		t.Fatalf("expected type to be omitted, got %#v", record["type"])
	}
}

func TestCreateFlowQueriesUserTasksToResolveRecordID(t *testing.T) {
	t.Helper()

	requestBodies := make([]map[string]any, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		var payload map[string]any
		if len(body) > 0 {
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
		}
		requestBodies = append(requestBodies, payload)

		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","tenant_access_token":"token-123","expire":7200}`))
		case "/open-apis/attendance/v1/user_flows/batch_create":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"msg":"","data":{"flow_records":[{"check_time":"1774065317","comment":"Syslog 导入上班打卡","creator_id":"user-001","location_name":"办公室","user_id":"user-001"}]}}`))
		case "/open-apis/attendance/v1/user_tasks/query":
			if got := r.URL.Query().Get("employee_type"); got != "employee_id" {
				t.Fatalf("expected employee_type=employee_id, got %q", got)
			}
			if got := r.URL.Query().Get("ignore_invalid_users"); got != "true" {
				t.Fatalf("expected ignore_invalid_users=true, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"user_task_results":[{"user_id":"user-001","day":20260321,"records":[{"check_in_record_id":"flow-lookup-001","check_in_record":{"user_id":"user-001","creator_id":"user-001","location_name":"办公室","check_time":"1774065317","comment":"Syslog 导入上班打卡","record_id":"flow-lookup-001"}}]}]}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewFeishuAttendanceHTTPClient(server.Client())
	client.baseURL = server.URL

	result, err := client.CreateFlow(context.Background(), FeishuAttendanceConfig{
		AppID:        "cli_123",
		AppSecret:    "secret_456",
		LocationName: "办公室",
		Timeout:      5 * time.Second,
	}, FeishuAttendanceCreateInput{
		EmployeeType: "employee_id",
		UserID:       "user-001",
		CreatorID:    "user-001",
		LocationName: "办公室",
		CheckTime:    "1774065317",
		Comment:      "Syslog 导入上班打卡",
	})
	if err != nil {
		t.Fatalf("expected create flow to succeed via user_tasks query, got %v", err)
	}
	if result.RecordID != "flow-lookup-001" {
		t.Fatalf("expected resolved record id flow-lookup-001, got %q", result.RecordID)
	}

	if len(requestBodies) != 3 {
		t.Fatalf("expected token request, create request, and query request, got %d requests", len(requestBodies))
	}

	queryPayload := requestBodies[2]
	if got := queryPayload["check_date_from"]; got != float64(20260321) {
		t.Fatalf("expected check_date_from 20260321, got %#v", got)
	}
	if got := queryPayload["check_date_to"]; got != float64(20260321) {
		t.Fatalf("expected check_date_to 20260321, got %#v", got)
	}

	userIDs, ok := queryPayload["user_ids"].([]any)
	if !ok || len(userIDs) != 1 || userIDs[0] != "user-001" {
		t.Fatalf("expected user_ids [user-001], got %#v", queryPayload["user_ids"])
	}
}

func TestSendTextMessageUsesUserIDReceiverAndReturnsMessageID(t *testing.T) {
	t.Helper()

	requestBodies := make([]map[string]any, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		var payload map[string]any
		if len(body) > 0 {
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
		}
		requestBodies = append(requestBodies, payload)

		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","tenant_access_token":"token-123","expire":7200}`))
		case "/open-apis/im/v1/messages":
			if got := r.URL.Query().Get("receive_id_type"); got != "user_id" {
				t.Fatalf("expected receive_id_type=user_id, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"message_id":"om_msg_001","msg_type":"text"}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewFeishuAttendanceHTTPClient(server.Client())
	client.baseURL = server.URL

	result, err := client.SendTextMessage(context.Background(), FeishuAttendanceConfig{
		AppID:        "cli_123",
		AppSecret:    "secret_456",
		LocationName: "办公室",
		Timeout:      5 * time.Second,
	}, FeishuSendMessageInput{
		ReceiveIDType: "user_id",
		ReceiveID:     "fs_emp_007",
		Text:          "打卡成功",
		UUID:          "notify-001",
	})
	if err != nil {
		t.Fatalf("expected send text message to succeed, got %v", err)
	}
	if result.MessageID != "om_msg_001" {
		t.Fatalf("expected message id om_msg_001, got %q", result.MessageID)
	}

	if len(requestBodies) != 2 {
		t.Fatalf("expected token request and send message request, got %d requests", len(requestBodies))
	}

	sendPayload := requestBodies[1]
	if got := sendPayload["receive_id"]; got != "fs_emp_007" {
		t.Fatalf("expected receive_id fs_emp_007, got %#v", got)
	}
	if got := sendPayload["msg_type"]; got != "text" {
		t.Fatalf("expected msg_type text, got %#v", got)
	}
	if got := sendPayload["uuid"]; got != "notify-001" {
		t.Fatalf("expected uuid notify-001, got %#v", got)
	}
	if got := sendPayload["content"]; got != "{\"text\":\"打卡成功\"}" {
		t.Fatalf("expected escaped text content, got %#v", got)
	}
}
