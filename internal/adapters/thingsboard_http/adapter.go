package thingsboardhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nms-agent/internal/adapters/base"
	"nms-agent/internal/models"
)

type Adapter struct {
	baseURL string
	http    *http.Client
	tokens  interface {
		GetThingsBoardToken(context.Context, string) (string, bool, error)
		SaveThingsBoardToken(context.Context, string, string) error
		MarkThingsBoardTokenUsed(context.Context, string) error
	}
	obs base.AdapterObserver
}
type telemetryPayload struct {
	TS     int64          `json:"ts"`
	Values map[string]any `json:"values"`
}

func NewAdapter(cfg map[string]any) (*Adapter, error) {
	baseURL, _ := cfg["base_url"].(string)
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("thingsboard_http requires config key 'base_url'")
	}
	return &Adapter{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}, nil
}
func (a *Adapter) SetThingsBoardTokenStore(store interface {
	GetThingsBoardToken(context.Context, string) (string, bool, error)
	SaveThingsBoardToken(context.Context, string, string) error
	MarkThingsBoardTokenUsed(context.Context, string) error
}) {
	a.tokens = store
}
func (a *Adapter) SetObserver(obs base.AdapterObserver) { a.obs = obs }

func (a *Adapter) SendBatch(ctx context.Context, batch []models.Telemetry) error {
	if a == nil || a.http == nil || a.tokens == nil {
		return errors.New("thingsboard_http adapter is not initialized")
	}
	byDevice := map[string]telemetryPayload{}
	for _, item := range batch {
		if item.DeviceID == "" || item.Metric == "" {
			continue
		}
		payload := byDevice[item.DeviceID]
		if payload.Values == nil {
			payload.TS = item.TS.UnixMilli()
			payload.Values = map[string]any{}
		}
		value, err := valueOf(item)
		if err != nil {
			return err
		}
		payload.Values[item.Metric] = value
		byDevice[item.DeviceID] = payload
	}
	for deviceID, payload := range byDevice {
		token, ok, err := a.tokens.GetThingsBoardToken(ctx, deviceID)
		if err != nil {
			return err
		}
		if !ok || token == "" {
			return fmt.Errorf("no ThingsBoard token for device %s", deviceID)
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/v1/"+token+"/telemetry", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := a.http.Do(req)
		if err != nil {
			return fmt.Errorf("ThingsBoard HTTP telemetry: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("ThingsBoard HTTP telemetry: status %s", resp.Status)
		}
		_ = a.tokens.MarkThingsBoardTokenUsed(ctx, deviceID)
	}
	if a.obs != nil {
		a.obs.Update(batch)
		a.obs.UpdateStatus("published", fmt.Sprintf("count=%d devices=%d", len(batch), len(byDevice)))
	}
	return nil
}

func valueOf(item models.Telemetry) (any, error) {
	if item.ValueType == "string" {
		if item.ValueString == nil {
			return nil, errors.New("string telemetry value is nil")
		}
		return *item.ValueString, nil
	}
	if item.ValueNumber == nil {
		return nil, errors.New("numeric telemetry value is nil")
	}
	return *item.ValueNumber, nil
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL, nil)
	if err != nil {
		return err
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
