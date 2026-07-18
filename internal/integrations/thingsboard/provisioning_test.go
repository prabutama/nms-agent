package thingsboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProvisioningClientProvisionDevice(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/provision" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"status":"SUCCESS","credentialsType":"ACCESS_TOKEN","credentialsValue":"token"}`))
	}))
	defer srv.Close()

	c := NewProvisioningClient(ProvisioningConfig{BaseURL: srv.URL, DeviceKey: "key", DeviceSecret: "secret"})
	token, err := c.ProvisionDevice(context.Background(), "d1")
	if err != nil {
		t.Fatalf("ProvisionDevice: %v", err)
	}
	if token != "token" {
		t.Fatalf("token=%q", token)
	}
	if got["deviceName"] != "d1" || got["provisionDeviceKey"] != "key" || got["provisionDeviceSecret"] != "secret" {
		t.Fatalf("request=%v", got)
	}
}

func TestProvisioningClientRejectsFailureStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"FAILURE"}`))
	}))
	defer srv.Close()

	c := NewProvisioningClient(ProvisioningConfig{BaseURL: srv.URL, DeviceKey: "key", DeviceSecret: "secret"})
	if _, err := c.ProvisionDevice(context.Background(), "d1"); err == nil {
		t.Fatalf("expected error")
	}
}
