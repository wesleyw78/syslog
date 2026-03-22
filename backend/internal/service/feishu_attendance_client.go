package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	feishuOpenAPIBaseURL = "https://open.feishu.cn"
	tokenRefreshSkew     = 30 * time.Minute
)

type FeishuAttendanceConfig struct {
	AppID             string
	AppSecret         string
	CreatorEmployeeID string
	LocationName      string
	Timeout           time.Duration
}

type FeishuAttendanceCreateInput struct {
	EmployeeType string
	UserID       string
	CreatorID    string
	LocationName string
	CheckTime    string
	Comment      string
	ExternalID   string
	IdempotentID string
	Type         int
}

type FeishuCreateFlowResult struct {
	RecordID     string
	StatusCode   int
	ResponseBody string
}

type FeishuDeleteFlowsResult struct {
	SuccessRecordIDs []string
	StatusCode       int
	ResponseBody     string
}

type FeishuSendMessageInput struct {
	ReceiveIDType string
	ReceiveID     string
	Text          string
	UUID          string
}

type FeishuSendMessageResult struct {
	MessageID    string
	StatusCode   int
	ResponseBody string
}

type FeishuAttendanceClient interface {
	CreateFlow(ctx context.Context, config FeishuAttendanceConfig, input FeishuAttendanceCreateInput) (*FeishuCreateFlowResult, error)
	DeleteFlows(ctx context.Context, config FeishuAttendanceConfig, recordIDs []string) (*FeishuDeleteFlowsResult, error)
	SendTextMessage(ctx context.Context, config FeishuAttendanceConfig, input FeishuSendMessageInput) (*FeishuSendMessageResult, error)
}

type FeishuAttendanceHTTPClient struct {
	baseURL    string
	httpClient *http.Client

	mu              sync.Mutex
	cachedToken     string
	cachedExpiresAt time.Time
	cachedAppID     string
	cachedSecret    string
}

func NewFeishuAttendanceHTTPClient(httpClient *http.Client) *FeishuAttendanceHTTPClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &FeishuAttendanceHTTPClient{
		baseURL:    feishuOpenAPIBaseURL,
		httpClient: httpClient,
	}
}

func (c *FeishuAttendanceHTTPClient) CreateFlow(ctx context.Context, config FeishuAttendanceConfig, input FeishuAttendanceCreateInput) (*FeishuCreateFlowResult, error) {
	token, err := c.tenantAccessToken(ctx, config)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"flow_records": []map[string]any{
			{
				"user_id":       input.UserID,
				"creator_id":    input.CreatorID,
				"location_name": input.LocationName,
				"check_time":    input.CheckTime,
				"comment":       input.Comment,
			},
		},
	}

	url := fmt.Sprintf("%s/open-apis/attendance/v1/user_flows/batch_create?employee_type=%s", c.baseURL, input.EmployeeType)
	responseBody, statusCode, err := c.doJSON(ctx, config.Timeout, http.MethodPost, url, map[string]string{
		"Authorization": "Bearer " + token,
	}, requestBody)
	result := &FeishuCreateFlowResult{
		StatusCode:   statusCode,
		ResponseBody: responseBody,
	}
	if err != nil {
		log.Printf(
			"feishu create flow failed: employee_type=%s user_id=%s external_id=%s status=%d err=%v response=%s",
			input.EmployeeType,
			input.UserID,
			input.ExternalID,
			statusCode,
			err,
			sanitizeFeishuResponseBody(responseBody),
		)
		return result, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			FlowRecords []struct {
				RecordID string `json:"record_id"`
			} `json:"flow_records"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		return result, fmt.Errorf("decode feishu create response: %w", err)
	}
	if response.Code != 0 {
		return result, fmt.Errorf("feishu create flow failed: code=%d msg=%s", response.Code, response.Msg)
	}

	if len(response.Data.FlowRecords) > 0 {
		result.RecordID = strings.TrimSpace(response.Data.FlowRecords[0].RecordID)
	}
	if result.RecordID == "" {
		recordID, lookupResponseBody, lookupErr := c.queryCreatedFlowRecordID(ctx, config, token, input)
		if strings.TrimSpace(lookupResponseBody) != "" {
			result.ResponseBody = combineFeishuResponseBodies(responseBody, lookupResponseBody)
		}
		if lookupErr != nil {
			return result, lookupErr
		}
		result.RecordID = recordID
	}

	log.Printf(
		"feishu create flow succeeded: employee_type=%s user_id=%s external_id=%s record_id=%s status=%d response=%s",
		input.EmployeeType,
		input.UserID,
		input.ExternalID,
		result.RecordID,
		statusCode,
		sanitizeFeishuResponseBody(responseBody),
	)
	return result, nil
}

func (c *FeishuAttendanceHTTPClient) queryCreatedFlowRecordID(ctx context.Context, config FeishuAttendanceConfig, token string, input FeishuAttendanceCreateInput) (string, string, error) {
	checkTimeUnix, err := strconv.ParseInt(strings.TrimSpace(input.CheckTime), 10, 64)
	if err != nil {
		return "", "", fmt.Errorf("parse feishu check_time for lookup: %w", err)
	}

	checkDate := time.Unix(checkTimeUnix, 0).In(time.Local)
	checkDateValue, err := strconv.Atoi(checkDate.Format("20060102"))
	if err != nil {
		return "", "", fmt.Errorf("format feishu check_date for lookup: %w", err)
	}

	requestBody := map[string]any{
		"user_ids":        []string{input.UserID},
		"check_date_from": checkDateValue,
		"check_date_to":   checkDateValue,
	}
	url := fmt.Sprintf("%s/open-apis/attendance/v1/user_tasks/query?employee_type=%s&ignore_invalid_users=true", c.baseURL, input.EmployeeType)
	responseBody, statusCode, err := c.doJSON(ctx, config.Timeout, http.MethodPost, url, map[string]string{
		"Authorization": "Bearer " + token,
	}, requestBody)
	if err != nil {
		log.Printf(
			"feishu query user tasks failed: employee_type=%s user_id=%s check_date=%d status=%d err=%v response=%s",
			input.EmployeeType,
			input.UserID,
			checkDateValue,
			statusCode,
			err,
			sanitizeFeishuResponseBody(responseBody),
		)
		return "", responseBody, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			UserTaskResults []struct {
				UserID  string `json:"user_id"`
				Day     int    `json:"day"`
				Records []struct {
					CheckInRecordID  string         `json:"check_in_record_id"`
					CheckInRecord    feishuUserFlow `json:"check_in_record"`
					CheckOutRecordID string         `json:"check_out_record_id"`
					CheckOutRecord   feishuUserFlow `json:"check_out_record"`
				} `json:"records"`
			} `json:"user_task_results"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		return "", responseBody, fmt.Errorf("decode feishu user tasks response: %w", err)
	}
	if response.Code != 0 {
		return "", responseBody, fmt.Errorf("feishu user tasks query failed: code=%d msg=%s", response.Code, response.Msg)
	}

	for _, task := range response.Data.UserTaskResults {
		for _, record := range task.Records {
			if recordID := matchFeishuQueriedRecord(input, record.CheckInRecordID, record.CheckInRecord); recordID != "" {
				log.Printf(
					"feishu query user tasks resolved record_id: employee_type=%s user_id=%s check_date=%d record_id=%s",
					input.EmployeeType,
					input.UserID,
					checkDateValue,
					recordID,
				)
				return recordID, responseBody, nil
			}
			if recordID := matchFeishuQueriedRecord(input, record.CheckOutRecordID, record.CheckOutRecord); recordID != "" {
				log.Printf(
					"feishu query user tasks resolved record_id: employee_type=%s user_id=%s check_date=%d record_id=%s",
					input.EmployeeType,
					input.UserID,
					checkDateValue,
					recordID,
				)
				return recordID, responseBody, nil
			}
		}
	}

	log.Printf(
		"feishu query user tasks missing matching record_id: employee_type=%s user_id=%s check_date=%d response=%s",
		input.EmployeeType,
		input.UserID,
		checkDateValue,
		sanitizeFeishuResponseBody(responseBody),
	)
	return "", responseBody, fmt.Errorf("feishu create flow missing record_id after user_tasks query")
}

func (c *FeishuAttendanceHTTPClient) DeleteFlows(ctx context.Context, config FeishuAttendanceConfig, recordIDs []string) (*FeishuDeleteFlowsResult, error) {
	token, err := c.tenantAccessToken(ctx, config)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/open-apis/attendance/v1/user_flows/batch_del", c.baseURL)
	responseBody, statusCode, err := c.doJSON(ctx, config.Timeout, http.MethodPost, url, map[string]string{
		"Authorization": "Bearer " + token,
	}, map[string]any{"record_ids": recordIDs})
	result := &FeishuDeleteFlowsResult{
		StatusCode:   statusCode,
		ResponseBody: responseBody,
	}
	if err != nil {
		log.Printf(
			"feishu delete flow failed: record_ids=%s status=%d err=%v response=%s",
			strings.Join(recordIDs, ","),
			statusCode,
			err,
			sanitizeFeishuResponseBody(responseBody),
		)
		return result, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			SuccessRecordIDs []string `json:"success_record_ids"`
			FailRecordIDs    []string `json:"fail_record_ids"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		return result, fmt.Errorf("decode feishu delete response: %w", err)
	}
	if response.Code != 0 {
		return result, fmt.Errorf("feishu delete flow failed: code=%d msg=%s", response.Code, response.Msg)
	}
	if len(response.Data.FailRecordIDs) > 0 {
		return result, fmt.Errorf("feishu delete flow returned failed record ids: %s", strings.Join(response.Data.FailRecordIDs, ","))
	}

	result.SuccessRecordIDs = append([]string(nil), response.Data.SuccessRecordIDs...)
	log.Printf(
		"feishu delete flow succeeded: requested_record_ids=%s success_record_ids=%s status=%d response=%s",
		strings.Join(recordIDs, ","),
		strings.Join(result.SuccessRecordIDs, ","),
		statusCode,
		sanitizeFeishuResponseBody(responseBody),
	)
	return result, nil
}

func (c *FeishuAttendanceHTTPClient) SendTextMessage(ctx context.Context, config FeishuAttendanceConfig, input FeishuSendMessageInput) (*FeishuSendMessageResult, error) {
	token, err := c.tenantAccessToken(ctx, config)
	if err != nil {
		return nil, err
	}

	content, err := json.Marshal(map[string]string{"text": input.Text})
	if err != nil {
		return nil, fmt.Errorf("encode feishu message content: %w", err)
	}

	requestBody := map[string]any{
		"receive_id": input.ReceiveID,
		"msg_type":   "text",
		"content":    string(content),
		"uuid":       input.UUID,
	}

	url := fmt.Sprintf("%s/open-apis/im/v1/messages?receive_id_type=%s", c.baseURL, input.ReceiveIDType)
	responseBody, statusCode, err := c.doJSON(ctx, config.Timeout, http.MethodPost, url, map[string]string{
		"Authorization": "Bearer " + token,
	}, requestBody)
	result := &FeishuSendMessageResult{
		StatusCode:   statusCode,
		ResponseBody: responseBody,
	}
	if err != nil {
		log.Printf(
			"feishu send message failed: receive_id_type=%s receive_id=%s status=%d err=%v response=%s",
			input.ReceiveIDType,
			input.ReceiveID,
			statusCode,
			err,
			sanitizeFeishuResponseBody(responseBody),
		)
		return result, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		return result, fmt.Errorf("decode feishu send message response: %w", err)
	}
	if response.Code != 0 {
		return result, fmt.Errorf("feishu send message failed: code=%d msg=%s", response.Code, response.Msg)
	}

	result.MessageID = strings.TrimSpace(response.Data.MessageID)
	log.Printf(
		"feishu send message succeeded: receive_id_type=%s receive_id=%s message_id=%s status=%d response=%s",
		input.ReceiveIDType,
		input.ReceiveID,
		result.MessageID,
		statusCode,
		sanitizeFeishuResponseBody(responseBody),
	)
	return result, nil
}

func (c *FeishuAttendanceHTTPClient) tenantAccessToken(ctx context.Context, config FeishuAttendanceConfig) (string, error) {
	c.mu.Lock()
	if c.cachedToken != "" &&
		time.Now().Before(c.cachedExpiresAt.Add(-tokenRefreshSkew)) &&
		c.cachedAppID == config.AppID &&
		c.cachedSecret == config.AppSecret {
		token := c.cachedToken
		c.mu.Unlock()
		log.Printf("feishu tenant token cache hit: app_id=%s expires_at=%s", config.AppID, c.cachedExpiresAt.UTC().Format(time.RFC3339))
		return token, nil
	}
	c.mu.Unlock()

	log.Printf("feishu tenant token request start: app_id=%s timeout=%s", config.AppID, config.Timeout)
	responseBody, statusCode, err := c.doJSON(ctx, config.Timeout, http.MethodPost, fmt.Sprintf("%s/open-apis/auth/v3/tenant_access_token/internal", c.baseURL), nil, map[string]string{
		"app_id":     config.AppID,
		"app_secret": config.AppSecret,
	})
	if err != nil {
		log.Printf(
			"feishu tenant token request failed: app_id=%s status=%d err=%v response=%s",
			config.AppID,
			statusCode,
			err,
			sanitizeFeishuResponseBody(responseBody),
		)
		return "", err
	}

	var response struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int64  `json:"expire"`
	}
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		return "", fmt.Errorf("decode feishu tenant token response: %w", err)
	}
	if response.Code != 0 {
		return "", fmt.Errorf("feishu tenant token failed: status=%d code=%d msg=%s", statusCode, response.Code, response.Msg)
	}
	if strings.TrimSpace(response.TenantAccessToken) == "" {
		return "", fmt.Errorf("feishu tenant token response missing token")
	}

	expiresAt := time.Now().Add(time.Duration(response.Expire) * time.Second)
	c.mu.Lock()
	c.cachedToken = response.TenantAccessToken
	c.cachedExpiresAt = expiresAt
	c.cachedAppID = config.AppID
	c.cachedSecret = config.AppSecret
	c.mu.Unlock()

	log.Printf(
		"feishu tenant token request succeeded: app_id=%s expires_at=%s response=%s",
		config.AppID,
		expiresAt.UTC().Format(time.RFC3339),
		sanitizeFeishuResponseBody(responseBody),
	)
	return response.TenantAccessToken, nil
}

func (c *FeishuAttendanceHTTPClient) doJSON(ctx context.Context, timeout time.Duration, method string, url string, headers map[string]string, body any) (string, int, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload, err := json.Marshal(body)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return "", 0, err
	}

	log.Printf(
		"feishu http request: method=%s url=%s timeout=%s headers=%s body=%s",
		method,
		url,
		timeout,
		sanitizeFeishuHeaders(headers),
		sanitizeFeishuBody(body),
	)
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", resp.StatusCode, readErr
	}
	log.Printf(
		"feishu http response: method=%s url=%s status=%d body=%s",
		method,
		url,
		resp.StatusCode,
		sanitizeFeishuResponseBody(string(responseBody)),
	)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return string(responseBody), resp.StatusCode, fmt.Errorf("unexpected feishu status: %d", resp.StatusCode)
	}

	return string(responseBody), resp.StatusCode, nil
}

func sanitizeFeishuHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return "{}"
	}

	sanitized := make(map[string]string, len(headers))
	for key, value := range headers {
		if strings.EqualFold(key, "authorization") {
			sanitized[key] = "Bearer ***"
			continue
		}
		sanitized[key] = value
	}

	payload, err := json.Marshal(sanitized)
	if err != nil {
		return `{"error":"marshal headers failed"}`
	}
	return string(payload)
}

func sanitizeFeishuBody(body any) string {
	if body == nil {
		return "{}"
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return `{"error":"marshal body failed"}`
	}
	return sanitizeFeishuResponseBody(string(payload))
}

var (
	appSecretPattern         = regexp.MustCompile(`("app_secret"\s*:\s*")([^"]*)(")`)
	tenantAccessTokenPattern = regexp.MustCompile(`("tenant_access_token"\s*:\s*")([^"]*)(")`)
	authorizationPattern     = regexp.MustCompile(`("Authorization"\s*:\s*")([^"]*)(")`)
)

func sanitizeFeishuResponseBody(raw string) string {
	sanitized := raw
	sanitized = appSecretPattern.ReplaceAllString(sanitized, `${1}***${3}`)
	sanitized = tenantAccessTokenPattern.ReplaceAllString(sanitized, `${1}***${3}`)
	sanitized = authorizationPattern.ReplaceAllString(sanitized, `${1}Bearer ***${3}`)
	return sanitized
}

type feishuUserFlow struct {
	UserID       string `json:"user_id"`
	CreatorID    string `json:"creator_id"`
	LocationName string `json:"location_name"`
	CheckTime    string `json:"check_time"`
	Comment      string `json:"comment"`
	RecordID     string `json:"record_id"`
}

func matchFeishuQueriedRecord(input FeishuAttendanceCreateInput, fallbackRecordID string, flow feishuUserFlow) string {
	if strings.TrimSpace(flow.UserID) != strings.TrimSpace(input.UserID) {
		return ""
	}
	if creatorID := strings.TrimSpace(input.CreatorID); creatorID != "" && strings.TrimSpace(flow.CreatorID) != creatorID {
		return ""
	}
	if locationName := strings.TrimSpace(input.LocationName); locationName != "" && strings.TrimSpace(flow.LocationName) != locationName {
		return ""
	}
	if checkTime := strings.TrimSpace(input.CheckTime); checkTime != "" && strings.TrimSpace(flow.CheckTime) != checkTime {
		return ""
	}
	if comment := strings.TrimSpace(input.Comment); comment != "" && strings.TrimSpace(flow.Comment) != comment {
		return ""
	}

	if recordID := strings.TrimSpace(flow.RecordID); recordID != "" {
		return recordID
	}
	return strings.TrimSpace(fallbackRecordID)
}

func combineFeishuResponseBodies(primary string, secondary string) string {
	primary = strings.TrimSpace(primary)
	secondary = strings.TrimSpace(secondary)
	switch {
	case primary == "":
		return secondary
	case secondary == "":
		return primary
	default:
		return fmt.Sprintf(`{"create_response":%s,"query_response":%s}`, primary, secondary)
	}
}
