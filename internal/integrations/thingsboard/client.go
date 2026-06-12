package thingsboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(cfg APIConfig) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		apiKey:  strings.TrimSpace(cfg.APIKey),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) enabled() bool { return c != nil && c.baseURL != "" && c.apiKey != "" }

func (c *Client) CheckAuth(ctx context.Context, assetID string) error {
	if !c.enabled() {
		return fmt.Errorf("thingsboard API config is incomplete")
	}
	_, err := c.GetRelations(ctx, assetID)
	return err
}

func (c *Client) GetRelations(ctx context.Context, assetID string) ([]Relation, error) {
	var out []Relation
	err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/api/relations?fromId="+url.QueryEscape(assetID)+"&fromType=ASSET", nil, &out)
	return out, err
}

func (c *Client) CreateRelation(ctx context.Context, assetID, deviceID, relationType string) error {
	body := map[string]any{
		"from": map[string]any{"entityType": "ASSET", "id": assetID},
		"to":   map[string]any{"entityType": "DEVICE", "id": deviceID},
		"type": relationType,
	}
	return c.doJSON(ctx, http.MethodPost, c.baseURL+"/api/relation", body, nil)
}

func (c *Client) CreateAlarm(ctx context.Context, alarm AlarmRequest) (*Alarm, error) {
	var out Alarm
	if err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/api/alarm", alarm, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ClearAlarm(ctx context.Context, alarmID string) error {
	return c.doJSON(ctx, http.MethodPost, c.baseURL+"/api/alarm/"+url.PathEscape(alarmID)+"/clear", nil, nil)
}

func (c *Client) GetAlarmsByEntity(ctx context.Context, entityType, entityID string) ([]Alarm, error) {
	var out []Alarm
	err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/api/alarm/"+url.PathEscape(entityType)+"/"+url.PathEscape(entityID)+"?pageSize=100&page=0&searchStatus=ACTIVE", nil, &out)
	return out, err
}

func (c *Client) SaveAssetServerAttributes(ctx context.Context, assetID string, attrs map[string]any) error {
	return c.doJSON(ctx, http.MethodPost, c.baseURL+"/api/plugins/telemetry/ASSET/"+url.PathEscape(assetID)+"/SERVER_SCOPE", attrs, nil)
}

func (c *Client) GetAssetServerAttributes(ctx context.Context, assetID string, keys []string) (map[string]any, error) {
	q := ""
	if len(keys) > 0 {
		q = "?keys=" + url.QueryEscape(strings.Join(keys, ","))
	}
	var out any
	err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/api/plugins/telemetry/ASSET/"+url.PathEscape(assetID)+"/values/attributes/SERVER_SCOPE"+q, nil, &out)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	switch v := out.(type) {
	case map[string]any:
		return v, nil
	case []any:
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key, _ := m["key"].(string)
			if key == "" {
				continue
			}
			result[key] = m["value"]
		}
	}
	return result, nil
}

func (c *Client) GetDeviceByName(ctx context.Context, name string, customerID string) (*DeviceInfo, error) {
	if customerID != "" {
		type devicePage struct {
			Data []DeviceInfo `json:"data"`
		}
		var page devicePage
		err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/api/customer/"+url.PathEscape(customerID)+"/deviceInfos?pageSize=1&page=0&textSearch="+url.QueryEscape(name), nil, &page)
		if err == nil && len(page.Data) > 0 {
			return &page.Data[0], nil
		}
	}
	return nil, fmt.Errorf("device %q not found", name)
}

func (c *Client) doJSON(ctx context.Context, method, rawURL string, body any, out any) error {
	if !c.enabled() {
		return fmt.Errorf("thingsboard API config is incomplete")
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, reader)
	if err != nil {
		return err
	}
	req.Header.Set("X-Authorization", "ApiKey "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("thingsboard API %s %s: %s", method, rawURL, strings.TrimSpace(string(data)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
