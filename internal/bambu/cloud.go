package bambu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	bambuAPIBase = "https://api.bambulab.com"
	// bambuUserAgent is the Bambu network-agent identifier documented by
	// OpenBambuAPI and used by Bambu Studio. It is not a browser User-Agent.
	bambuUserAgent    = "bambu_network_agent/01.09.05.01"
	cloudMQTTBrokerUS = "us.mqtt.bambulab.com"

	// cloudBlockedMessage is shown when Bambu's edge returns an HTML page
	// (Cloudflare block) or any other non-JSON body, instead of the API.
	cloudBlockedMessage = "Bambu Cloud rejected the request (blocked by Bambu's Cloudflare). Use LAN mode with IP, serial and Access Code."
)

// CloudClient handles communication with the Bambu Cloud API.
type CloudClient struct {
	httpClient *http.Client
	// baseURL is the API origin. Production uses bambuAPIBase; tests point it
	// at an httptest server.
	baseURL string
}

// NewCloudClient creates a new Bambu Cloud API client.
func NewCloudClient() *CloudClient {
	return &CloudClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: bambuAPIBase,
	}
}

// LoginResponse represents the response from the Bambu login API.
type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	LoginType    string `json:"loginType"` // "", "verifyCode", "tfa"
}

// CloudDevice represents a printer from the Bambu cloud device list.
type CloudDevice struct {
	DevID          string  `json:"dev_id"`
	Name           string  `json:"name"`
	Online         bool    `json:"online"`
	PrintStatus    string  `json:"print_status"`
	DevModelName   string  `json:"dev_model_name"`
	DevProductName string  `json:"dev_product_name"`
	DevAccessCode  string  `json:"dev_access_code"`
	NozzleDiameter float64 `json:"nozzle_diameter"`
}

// UserPreference holds user info including the UID needed for MQTT.
// The uid field comes back as a JSON number from the Bambu API.
type UserPreference struct {
	UID json.Number `json:"uid"`
}

// Login authenticates with Bambu Cloud using email and password.
// Returns the login response. If LoginType is "verifyCode", the caller
// must request a verification code and call LoginWithCode.
func (c *CloudClient) Login(email, password string) (*LoginResponse, error) {
	body := map[string]string{
		"account":  email,
		"password": password,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal login request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/v1/user-service/user/login", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create login request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	respBody, err := readAPI("login", resp)
	if err != nil {
		return nil, err
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return nil, fmt.Errorf("parse login response: %w", err)
	}

	return &loginResp, nil
}

// RequestVerifyCode requests a verification code to be sent via email.
func (c *CloudClient) RequestVerifyCode(email string) error {
	body := map[string]string{
		"email": email,
		"type":  "codeLogin",
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal verify request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/v1/user-service/user/sendemail/code", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create verify request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("verify code request failed: %w", err)
	}
	if _, err := readAPI("verify code request", resp); err != nil {
		return err
	}

	return nil
}

// LoginWithCode completes login using email and verification code.
func (c *CloudClient) LoginWithCode(email, code string) (*LoginResponse, error) {
	body := map[string]string{
		"account": email,
		"code":    code,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal code login request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/v1/user-service/user/login", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create code login request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("code login request failed: %w", err)
	}
	respBody, err := readAPI("code login", resp)
	if err != nil {
		return nil, err
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return nil, fmt.Errorf("parse code login response: %w", err)
	}

	return &loginResp, nil
}

// GetUsername fetches the MQTT username (uid) from user preferences.
// Returns the username in format "u_{uid}".
func (c *CloudClient) GetUsername(token string) (string, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/v1/design-user-service/my/preference", nil)
	if err != nil {
		return "", fmt.Errorf("create preference request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("preference request failed: %w", err)
	}
	respBody, err := readAPI("preference request", resp)
	if err != nil {
		return "", err
	}

	var pref UserPreference
	if err := json.Unmarshal(respBody, &pref); err != nil {
		return "", fmt.Errorf("parse preference response: %w", err)
	}

	if pref.UID.String() == "" || pref.UID.String() == "0" {
		return "", fmt.Errorf("empty UID in preference response")
	}

	return "u_" + pref.UID.String(), nil
}

// GetDevices fetches the list of printers bound to the account.
func (c *CloudClient) GetDevices(token string) ([]CloudDevice, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/v1/iot-service/api/user/bind", nil)
	if err != nil {
		return nil, fmt.Errorf("create devices request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("devices request failed: %w", err)
	}
	respBody, err := readAPI("devices request", resp)
	if err != nil {
		return nil, err
	}

	// The response wraps devices in a "devices" field
	var wrapper struct {
		Devices []CloudDevice `json:"devices"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		// Try parsing as bare array
		var devices []CloudDevice
		if err2 := json.Unmarshal(respBody, &devices); err2 != nil {
			return nil, fmt.Errorf("parse devices response: %w (also tried array: %w)", err, err2)
		}
		return devices, nil
	}

	return wrapper.Devices, nil
}

// RefreshToken uses the refresh token to get a new access token.
func (c *CloudClient) RefreshToken(refreshToken string) (*LoginResponse, error) {
	body := map[string]string{
		"refreshToken": refreshToken,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal refresh request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/v1/user-service/user/refreshtoken", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create refresh request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	respBody, err := readAPI("refresh", resp)
	if err != nil {
		return nil, err
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return nil, fmt.Errorf("parse refresh response: %w", err)
	}

	return &loginResp, nil
}

// CloudMQTTBroker returns the cloud MQTT broker address for the given region.
func CloudMQTTBroker() string {
	return cloudMQTTBrokerUS
}

// setHeaders sets the required headers for Bambu API requests.
func (c *CloudClient) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", bambuUserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// readAPI reads a Bambu HTTP response. HTML and other non-JSON bodies — the
// Cloudflare block page in particular — become a short error. JSON error
// bodies keep Bambu's message and never include the raw document when it
// is not JSON.
func readAPI(action string, resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", action, err)
	}
	if nonJSONBody(body) {
		slog.Warn("bambu cloud returned a non-json response",
			"action", action,
			"status", resp.StatusCode,
			"content_type", resp.Header.Get("Content-Type"),
		)
		// User-facing copy shown in the printer UI. A trailing period is intentional.
		return nil, errors.New(cloudBlockedMessage) //nolint:staticcheck // ST1005: sentence is shown to the user as-is
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s failed (status %d): %s", action, resp.StatusCode, jsonAPIMessage(body))
	}
	return body, nil
}

func nonJSONBody(body []byte) bool {
	b := bytes.TrimSpace(body)
	if len(b) == 0 {
		return false
	}
	if (b[0] == '{' || b[0] == '[') && json.Valid(b) {
		return false
	}
	return true
}

func jsonAPIMessage(body []byte) string {
	var probe struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &probe); err == nil {
		msg := strings.TrimSpace(probe.Message)
		if msg == "" {
			msg = strings.TrimSpace(probe.Error)
		}
		if msg != "" && !looksLikeMarkup(msg) {
			return truncateOneLine(msg, 200)
		}
	}
	fallback := truncateOneLine(string(bytes.TrimSpace(body)), 200)
	if looksLikeMarkup(fallback) {
		return cloudBlockedMessage
	}
	return fallback
}

func looksLikeMarkup(s string) bool {
	trimmed := strings.TrimSpace(s)
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(trimmed, "<") || strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype")
}

func truncateOneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
