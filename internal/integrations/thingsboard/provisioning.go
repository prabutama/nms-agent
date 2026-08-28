package thingsboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ProvisioningConfig struct {
	BaseURL      string
	DeviceKey    string
	DeviceSecret string
}

type ProvisioningClient struct {
	baseURL      string
	deviceKey    string
	deviceSecret string
	http         *http.Client
}

type provisioningRequest struct {
	DeviceName            string `json:"deviceName"`
	ProvisionDeviceKey    string `json:"provisionDeviceKey"`
	ProvisionDeviceSecret string `json:"provisionDeviceSecret"`
}

type provisioningResponse struct {
	Status           string `json:"status"`
	CredentialsType  string `json:"credentialsType"`
	CredentialsValue string `json:"credentialsValue"`
	CredentialsID    string `json:"credentialsId"`
}

func NewProvisioningClient(cfg ProvisioningConfig) *ProvisioningClient {
	return &ProvisioningClient{
		baseURL:      strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		deviceKey:    strings.TrimSpace(cfg.DeviceKey),
		deviceSecret: strings.TrimSpace(cfg.DeviceSecret),
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *ProvisioningClient) ProvisionDevice(ctx context.Context, deviceName string) (string, error) {
	if c == nil || c.baseURL == "" || c.deviceKey == "" || c.deviceSecret == "" {
		return "", fmt.Errorf("thingsboard provisioning config is incomplete")
	}
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" {
		return "", fmt.Errorf("device name is required")
	}
	reqBody := provisioningRequest{DeviceName: deviceName, ProvisionDeviceKey: c.deviceKey, ProvisionDeviceSecret: c.deviceSecret}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/provision", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("thingsboard provisioning: %s", strings.TrimSpace(string(data)))
	}
	var out provisioningResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if !strings.EqualFold(out.Status, "SUCCESS") {
		return "", fmt.Errorf("thingsboard provisioning failed: status=%s", out.Status)
	}
	if !strings.EqualFold(out.CredentialsType, "ACCESS_TOKEN") {
		return "", fmt.Errorf("thingsboard provisioning returned unsupported credentials type %q", out.CredentialsType)
	}
	token := strings.TrimSpace(out.CredentialsValue)
	if token == "" {
		token = strings.TrimSpace(out.CredentialsID)
	}
	if token == "" {
		return "", fmt.Errorf("thingsboard provisioning returned empty access token")
	}
	return token, nil
}
