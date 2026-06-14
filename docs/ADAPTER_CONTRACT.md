# Adapter Contract & Development Guide

## 1. Rules

1. Adapter must NOT collect data.
2. Adapter must NOT modify the queue directly.
3. Adapter returns success or failure to the queue worker.
4. Adapter must NOT drop telemetry silently.
5. Adapter must call `a.obs.Update(batch)` after a successful send, and `a.obs.UpdateStatus(status, detail)` on connection changes.
6. Adapter must implement at minimum `base.Adapter` (`SendBatch`).
7. Adapter must accept `config map[string]any` in the constructor.

---

## 2. Directory Structure

```
internal/adapters/<name>/
├── adapter.go       # adapter implementation
├── adapter_test.go  # unit tests (optional)
└── route_test.go    # routing tests (optional)
```

Package naming convention:
- Directory `generic_mqtt/` → `package genericmqtt`
- Directory `thingsboard_mqtt/` → `package thingsboardmqtt`
- Directory `my_adapter/` → `package myadapter`

---

## 3. Available Interfaces (from `base/port.go`)

```go
// Required
type Adapter interface {
    SendBatch(ctx context.Context, batch []models.Telemetry) error
}

// Optional
type HealthChecker interface {
    HealthCheck(ctx context.Context) error
}

// Optional — for resource cleanup
type Closable interface {
    Close() error
}

// Optional — for observer hub (MUST be wired!)
type ObserverSetter interface {
    SetObserver(hub AdapterObserver)
}

// Observer that must be stored and called
type AdapterObserver interface {
    Update(batch []models.Telemetry)
    UpdateStatus(status string, details string)
}
```

### REQUIRED: Observer Pattern

Every adapter MUST store a `base.AdapterObserver` and call:

| Method | When to call |
|--------|-------------|
| `a.obs.Update(batch)` | After a batch is sent successfully |
| `a.obs.UpdateStatus("published", "count=N")` | After successful publish |
| `a.obs.UpdateStatus("connect_failed", err.Error())` | When connection fails |
| `a.obs.UpdateStatus("not_connected", "broker unreachable")` | When broker is unreachable |

---

## 4. Data Model (`internal/models/telemetry.go`)

```go
type Telemetry struct {
    DeviceID    string
    Metric      string
    TS          time.Time
    ValueType   string            // "number" | "string"
    ValueNumber *float64
    ValueString *string
    Tags        map[string]string
}
```

The adapter receives `[]models.Telemetry` via `SendBatch`. You can marshal to JSON, transform, or send directly.

---

## 5. Utility Functions (from `internal/adapters/base/`)

| Function | Signature | Purpose |
|----------|-----------|---------|
| `GetOutputLocation` | `func() *time.Location` | Output timezone for timestamps |
| `FormatValue` | `func(t models.Telemetry) string` | Format value as string |
| `FormatTags` | `func(tags map[string]string) string` | Format tags as string |
| `FormatTS` | `func(ts time.Time) string` | Format timestamp as RFC3339 |

---

## 6. Implementation Templates

### Minimal (only SendBatch)

```go
package myadapter

import (
    "context"
    "errors"
    "fmt"

    "nms-agent/internal/adapters/base"
    "nms-agent/internal/models"
)

type myConfig struct {
    APIKey string
}

type MyAdapter struct {
    cfg myConfig
    obs base.AdapterObserver
}

func NewAdapter(cfg map[string]any) (*MyAdapter, error) {
    c := myConfig{}
    if v, ok := cfg["api_key"].(string); ok {
        c.APIKey = v
    }
    if c.APIKey == "" {
        return nil, errors.New("myadapter requires config key 'api_key'")
    }
    return &MyAdapter{cfg: c}, nil
}

func (a *MyAdapter) SendBatch(ctx context.Context, batch []models.Telemetry) error {
    // send data to the platform
    for _, t := range batch {
        _ = t
    }
    if a.obs != nil {
        a.obs.Update(batch)
        a.obs.UpdateStatus("published", fmt.Sprintf("count=%d", len(batch)))
    }
    return nil
}
```

### With HealthCheck + Close

```go
func (a *MyAdapter) HealthCheck(ctx context.Context) error {
    // ping the platform
    return nil
}

func (a *MyAdapter) Close() error {
    // cleanup resources
    return nil
}
```

### Required: Observer Setter

```go
func (a *MyAdapter) SetObserver(hub base.AdapterObserver) {
    a.obs = hub
}
```

---

## 7. Config Parsing Pattern

Use a separate struct for config:

```go
type myConfig struct {
    Field1 string
    Field2 int
    Field3 bool
}

func parseConfig(cfg map[string]any) (myConfig, error) {
    c := myConfig{
        Field2: 60,  // default value
        Field3: true,
    }
    if cfg == nil {
        return c, errors.New("config is required")
    }
    if v, ok := cfg["field1"].(string); ok && v != "" {
        c.Field1 = strings.TrimSpace(v)
    }
    if v, ok := cfg["field2"].(int); ok {
        c.Field2 = v
    } else if v, ok := cfg["field2"].(float64); ok {
        c.Field2 = int(v)
    }
    if v, ok := cfg["field3"].(bool); ok {
        c.Field3 = v
    }
    if c.Field1 == "" {
        return c, errors.New("config key 'field1' is required")
    }
    return c, nil
}
```

---

## 8. Registering in the Factory

Edit `internal/adapters/factory.go`:

```go
import (
    "nms-agent/internal/adapters/generic_mqtt"
    "nms-agent/internal/adapters/myadapter"            // <-- add import
    "nms-agent/internal/adapters/thingsboard_mqtt"
    "nms-agent/internal/adapters/tui"
)

func NewAdapter(name string, config map[string]any) (Adapter, error) {
    switch name {
    case "tui":
        return tui.NewAdapter(config)
    case "generic_mqtt":
        return genericmqtt.NewAdapter(config)
    case "my_adapter":                                  // <-- add case
        return myadapter.NewAdapter(config)
    default:
        return nil, fmt.Errorf("unknown adapter %q (supported: tui, generic_mqtt, my_adapter)", name)
    }
}
```

Also update `factory_test.go` to cover the new adapter.

---

## 9. Testing

Write unit tests with a fake observer:

```go
package myadapter

import (
    "context"
    "testing"
    "time"

    "nms-agent/internal/models"
)

type fakeObserver struct {
    updates      int
    lastStatus   string
    lastDetail   string
}

func (f *fakeObserver) Update(batch []models.Telemetry) {
    f.updates++
}
func (f *fakeObserver) UpdateStatus(status, detail string) {
    f.lastStatus = status
    f.lastDetail = detail
}

func TestMyAdapter_SendBatch(t *testing.T) {
    obs := &fakeObserver{}
    a := &MyAdapter{cfg: myConfig{APIKey: "test-key"}}
    a.SetObserver(obs)

    batch := []models.Telemetry{
        {DeviceID: "d1", Metric: "test.metric", ValueType: "number", ValueNumber: floatPtr(42), TS: time.Now().UTC()},
    }

    if err := a.SendBatch(context.Background(), batch); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if obs.updates != 1 {
        t.Fatalf("expected 1 observer update, got %d", obs.updates)
    }
}

func floatPtr(v float64) *float64 { return &v }
```

---

## 10. New Adapter Checklist

- [ ] Create directory `internal/adapters/<name>/`
- [ ] Implement `base.Adapter` (`SendBatch`)
- [ ] Implement `base.ObserverSetter` (`SetObserver`)
- [ ] Store `obs base.AdapterObserver`
- [ ] Call `obs.Update(batch)` after success
- [ ] Call `obs.UpdateStatus()` on important events
- [ ] Optionally: `base.HealthChecker`, `base.Closable`
- [ ] Parse config from `map[string]any`, return error if mandatory keys are missing
- [ ] Register in `internal/adapters/factory.go`
- [ ] Update `factory_test.go`
- [ ] Verify: `go build ./internal/adapters/...`
- [ ] Verify: `go test ./internal/adapters/`

---

## 11. References

| File | Contents |
|------|----------|
| `docs/DATA_CONTRACT.md` | Canonical telemetry format details |
| `docs/CONFIG_SCHEMA.md` | YAML config for adapters |
| `docs/ARCHITECTURE.md` | Hexagonal architecture rules |
| `internal/adapters/base/port.go` | Interface definitions |
| `internal/adapters/base/format.go` | Utility formatting |
| `internal/adapters/base/output_timezone.go` | Timezone management |
| `internal/adapters/terminal/adapter.go` | Minimal example |
| `internal/adapters/generic_mqtt/adapter.go` | Full example with health check |
| `internal/adapters/factory.go` | Adapter registration |
