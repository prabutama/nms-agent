package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"nms-agent/internal/collectors"
	"nms-agent/internal/config"
	"nms-agent/internal/profiles"

	yaml "gopkg.in/yaml.v3"
)

func runDevice(args []string) int {
	if len(args) < 1 {
		usage()
		return 2
	}
	switch args[0] {
	case "list":
		return runDeviceList(args[1:])
	case "add":
		return runDeviceAdd(args[1:])
	case "update":
		return runDeviceUpdate(args[1:])
	case "remove":
		return runDeviceRemove(args[1:])
	case "test":
		return runDeviceTest(args[1:])
	default:
		usage()
		return 2
	}
}

type triBool struct {
	set   bool
	value bool
}

func (t *triBool) String() string {
	if t == nil || !t.set {
		return ""
	}
	if t.value {
		return "true"
	}
	return "false"
}

func (t *triBool) Set(s string) error {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "true", "1", "yes", "y", "on":
		t.set = true
		t.value = true
		return nil
	case "false", "0", "no", "n", "off":
		t.set = true
		t.value = false
		return nil
	default:
		return fmt.Errorf("invalid bool %q", s)
	}
}

func findDeviceFileByID(devicesDir, id string) (string, error) {
	entries, err := os.ReadDir(devicesDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		p := filepath.Join(devicesDir, e.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var d config.Device
		if err := yaml.Unmarshal(b, &d); err != nil {
			continue
		}
		if strings.TrimSpace(d.ID) == id {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}

func runDeviceList(args []string) int {
	fs := flag.NewFlagSet("device list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	loaded, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(loaded); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	devs := append([]config.Device(nil), loaded.Devices...)
	sort.Slice(devs, func(i, j int) bool { return devs[i].ID < devs[j].ID })

	fmt.Fprintf(os.Stdout, "devices=%d\n", len(devs))
	if len(devs) == 0 {
		return 0
	}

	// Stable, greppable tabular output.
	fmt.Fprintln(os.Stdout, "id\taddress\tvendor\tmodel\tsnmp\ticmp")
	for _, d := range devs {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%s\t%t\t%t\n",
			d.ID,
			d.Address,
			d.Vendor,
			d.Model,
			d.SNMP.Enabled,
			d.ICMP.Enabled,
		)
	}
	return 0
}

func runDeviceAdd(args []string) int {
	fs := flag.NewFlagSet("device add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	id := fs.String("id", "", "Device ID (unique)")
	address := fs.String("address", "", "Device address (IP/host)")
	vendor := fs.String("vendor", "", "Device vendor (for profile selection)")
	model := fs.String("model", "", "Device model (for profile selection)")
	snmpEnabled := fs.Bool("snmp", true, "Enable SNMP collection")
	icmpEnabled := fs.Bool("icmp", true, "Enable ICMP collection")
	interactive := fs.Bool("interactive", false, "Force interactive wizard mode")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	*id = strings.TrimSpace(*id)
	*address = strings.TrimSpace(*address)
	*vendor = strings.TrimSpace(*vendor)
	*model = strings.TrimSpace(*model)

	if *id == "" || *address == "" || *vendor == "" || *model == "" {
		if *interactive || isInteractiveTerminal() {
			return runDeviceAddInteractive(configPath, id, address, vendor, model, snmpEnabled, icmpEnabled)
		}
		fmt.Fprintln(os.Stderr, "id, address, vendor, and model are required (use --interactive for wizard)")
		return 2
	}

	loaded, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(loaded); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	for _, d := range loaded.Devices {
		if d.ID == *id {
			fmt.Fprintln(os.Stderr, "device id already exists: "+*id)
			return 1
		}
	}

	absCfg, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	baseDir := filepath.Dir(absCfg)
	devicesDir := config.ResolvePath(baseDir, loaded.Root.Paths.DevicesDir)
	if err := os.MkdirAll(devicesDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir devices_dir: "+err.Error())
		return 1
	}

	outPath := filepath.Join(devicesDir, *id+".yml")
	if _, err := os.Stat(outPath); err == nil {
		fmt.Fprintln(os.Stderr, "device file already exists: "+outPath)
		return 1
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "stat device file: "+err.Error())
		return 1
	}

	dev := config.Device{
		ID:      *id,
		Address: *address,
		Vendor:  *vendor,
		Model:   *model,
		SNMP:    config.DeviceSNMP{Enabled: *snmpEnabled},
		ICMP:    config.DeviceICMP{Enabled: *icmpEnabled},
	}
	b, err := yaml.Marshal(dev)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal device: "+err.Error())
		return 1
	}

	// Atomic write: write temp file in same dir then rename.
	tmpPath := filepath.Join(devicesDir, fmt.Sprintf(".%s.%d.tmp", *id, time.Now().UnixNano()))
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write temp device file: "+err.Error())
		return 1
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		fmt.Fprintln(os.Stderr, "rename temp device file: "+err.Error())
		return 1
	}

	// Re-validate full config after write to ensure loader can read it.
	if err := config.ValidateFiles(absCfg); err != nil {
		_ = os.Remove(outPath)
		fmt.Fprintln(os.Stderr, "config validation after write failed (rolled back new device file): "+err.Error())
		return 1
	}

	fmt.Fprintln(os.Stdout, "device added: "+outPath)
	return 0
}

func isInteractiveTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// sanitizeInput cleans raw reader input for safe use as device fields.
func sanitizeInput(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, "\r", "")
	var out strings.Builder
	out.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 32 && r < 127:
			out.WriteRune(r)
		case r == '\t':
			out.WriteString(" ")
		}
	}
	return strings.TrimSpace(out.String())
}

// validateDeviceID checks that id is safe for filename and config use.
func validateDeviceID(id string) error {
	if id == "" {
		return errors.New("device id is required")
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
			return fmt.Errorf("device id contains invalid character: %q", string(r))
		}
	}
	return nil
}

// validateAddress checks that address is a valid IP or hostname.
func validateAddress(addr string) error {
	if addr == "" {
		return errors.New("address is required")
	}
	// Basic IP validation (IPv4/IPv6).
	if net.ParseIP(addr) != nil {
		return nil
	}
	// Basic hostname validation.
	if len(addr) > 253 {
		return errors.New("address hostname too long")
	}
	labels := strings.Split(addr, ".")
	for _, l := range labels {
		if len(l) == 0 || len(l) > 63 {
			return fmt.Errorf("address hostname has invalid label: %q", l)
		}
		for _, r := range l {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
				return fmt.Errorf("address hostname has invalid character: %q", string(r))
			}
		}
	}
	return nil
}

// validateVendorModel checks vendor and model fields.
func validateVendorModel(vendor, model string) error {
	if vendor == "" {
		return errors.New("vendor is required")
	}
	if model == "" {
		return errors.New("model is required")
	}
	for _, r := range vendor {
		if r < 32 || (r > 126 && r != '\t') {
			return fmt.Errorf("vendor contains invalid character: %q", string(r))
		}
	}
	for _, r := range model {
		if r < 32 || (r > 126 && r != '\t') {
			return fmt.Errorf("model contains invalid character: %q", string(r))
		}
	}
	return nil
}

func runDeviceAddInteractive(configPath *string, id, address, vendor, model *string, snmpEnabled, icmpEnabled *bool) int {
	scanner := bufio.NewScanner(os.Stdin)

	prompt := func(label string) string {
		for {
			fmt.Fprintf(os.Stderr, "%s: ", label)
			if !scanner.Scan() {
				fmt.Fprintln(os.Stderr, "  input stream ended")
				return ""
			}
			input := scanner.Text()
			cleaned := strings.TrimSpace(input)
			if cleaned == "" {
				fmt.Fprintln(os.Stderr, "  cannot be empty, try again")
				continue
			}
			return cleaned
		}
	}

	promptBool := func(label string, defaultVal bool) bool {
		def := "Y"
		if !defaultVal {
			def = "N"
		}
		for {
			fmt.Fprintf(os.Stderr, "%s [%s]: ", label, def)
			if !scanner.Scan() {
				fmt.Fprintln(os.Stderr, "  input stream ended")
				return defaultVal
			}
			input := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if input == "" {
				return defaultVal
			}
			switch input {
			case "y", "yes", "1", "on":
				return true
			case "n", "no", "0", "off":
				return false
			default:
				fmt.Fprintln(os.Stderr, "  please enter Y or N")
			}
		}
	}

	*id = prompt("Device ID")
	if *id == "" {
		fmt.Fprintln(os.Stderr, "cancelled: input stream ended")
		return 2
	}
	if err := validateDeviceID(*id); err != nil {
		fmt.Fprintf(os.Stderr, "invalid device id: %v\n", err)
		return 2
	}

	*address = prompt("Address / IP")
	if *address == "" {
		fmt.Fprintln(os.Stderr, "cancelled: input stream ended")
		return 2
	}
	if err := validateAddress(*address); err != nil {
		fmt.Fprintf(os.Stderr, "invalid address: %v\n", err)
		return 2
	}

	*vendor = prompt("Vendor")
	if *vendor == "" {
		fmt.Fprintln(os.Stderr, "cancelled: input stream ended")
		return 2
	}

	*model = prompt("Model")
	if *model == "" {
		fmt.Fprintln(os.Stderr, "cancelled: input stream ended")
		return 2
	}

	*snmpEnabled = promptBool("Enable SNMP", true)
	*icmpEnabled = promptBool("Enable ICMP", true)

	fmt.Fprintln(os.Stderr, "\n--- Summary ---")
	fmt.Fprintf(os.Stderr, "ID:      %s\n", *id)
	fmt.Fprintf(os.Stderr, "Address: %s\n", *address)
	fmt.Fprintf(os.Stderr, "Vendor:  %s\n", *vendor)
	fmt.Fprintf(os.Stderr, "Model:   %s\n", *model)
	fmt.Fprintf(os.Stderr, "SNMP:    %v\n", *snmpEnabled)
	fmt.Fprintf(os.Stderr, "ICMP:    %v\n", *icmpEnabled)
	fmt.Fprintln(os.Stderr, "---------------")

	if !promptBool("Save device?", true) {
		fmt.Fprintln(os.Stderr, "cancelled")
		return 2
	}

	loaded, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(loaded); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	for _, d := range loaded.Devices {
		if d.ID == *id {
			fmt.Fprintln(os.Stderr, "device id already exists: "+*id)
			return 1
		}
	}

	absCfg, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	baseDir := filepath.Dir(absCfg)
	devicesDir := config.ResolvePath(baseDir, loaded.Root.Paths.DevicesDir)
	if err := os.MkdirAll(devicesDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir devices_dir: "+err.Error())
		return 1
	}

	outPath := filepath.Join(devicesDir, *id+".yml")
	if _, err := os.Stat(outPath); err == nil {
		fmt.Fprintln(os.Stderr, "device file already exists: "+outPath)
		return 1
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "stat device file: "+err.Error())
		return 1
	}

	dev := config.Device{
		ID:      *id,
		Address: *address,
		Vendor:  *vendor,
		Model:   *model,
		SNMP:    config.DeviceSNMP{Enabled: *snmpEnabled},
		ICMP:    config.DeviceICMP{Enabled: *icmpEnabled},
	}
	b, err := yaml.Marshal(dev)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal device: "+err.Error())
		return 1
	}

	tmpPath := filepath.Join(devicesDir, fmt.Sprintf(".%s.%d.tmp", *id, time.Now().UnixNano()))
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write temp device file: "+err.Error())
		return 1
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		fmt.Fprintln(os.Stderr, "rename temp device file: "+err.Error())
		return 1
	}

	if err := config.ValidateFiles(absCfg); err != nil {
		_ = os.Remove(outPath)
		fmt.Fprintln(os.Stderr, "config validation after write failed (rolled back new device file): "+err.Error())
		return 1
	}

	fmt.Fprintln(os.Stdout, "device added: "+outPath)
	return 0
}

func runDeviceUpdate(args []string) int {
	fs := flag.NewFlagSet("device update", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	id := fs.String("id", "", "Device ID (required)")
	address := fs.String("address", "", "New address (optional)")
	vendor := fs.String("vendor", "", "New vendor (optional)")
	model := fs.String("model", "", "New model (optional)")
	var snmp triBool
	var icmp triBool
	fs.Var(&snmp, "snmp", "Enable SNMP collection (true/false)")
	fs.Var(&icmp, "icmp", "Enable ICMP collection (true/false)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	*id = strings.TrimSpace(*id)
	if *id == "" {
		fmt.Fprintln(os.Stderr, "id is required")
		return 2
	}
	*address = strings.TrimSpace(*address)
	*vendor = strings.TrimSpace(*vendor)
	*model = strings.TrimSpace(*model)

	absCfg, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	loaded, err := config.LoadFromFile(absCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(loaded); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	devicesDir := config.ResolvePath(filepath.Dir(absCfg), loaded.Root.Paths.DevicesDir)
	path, err := findDeviceFileByID(devicesDir, *id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "device not found: "+*id)
			return 1
		}
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	oldBytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	var dev config.Device
	if err := yaml.Unmarshal(oldBytes, &dev); err != nil {
		fmt.Fprintln(os.Stderr, "invalid device file: "+err.Error())
		return 1
	}
	if strings.TrimSpace(dev.ID) != *id {
		fmt.Fprintln(os.Stderr, "device file id mismatch")
		return 1
	}

	if *address != "" {
		dev.Address = *address
	}
	if *vendor != "" {
		dev.Vendor = *vendor
	}
	if *model != "" {
		dev.Model = *model
	}
	if snmp.set {
		dev.SNMP.Enabled = snmp.value
	}
	if icmp.set {
		dev.ICMP.Enabled = icmp.value
	}

	// Basic required fields.
	dev.Address = strings.TrimSpace(dev.Address)
	dev.Vendor = strings.TrimSpace(dev.Vendor)
	dev.Model = strings.TrimSpace(dev.Model)
	if dev.Address == "" || dev.Vendor == "" || dev.Model == "" {
		fmt.Fprintln(os.Stderr, "address, vendor, and model must be non-empty")
		return 2
	}

	newBytes, err := yaml.Marshal(dev)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal device: "+err.Error())
		return 1
	}

	// Atomic replace.
	tmpPath := filepath.Join(devicesDir, fmt.Sprintf(".%s.%d.tmp", *id, time.Now().UnixNano()))
	if err := os.WriteFile(tmpPath, newBytes, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write temp device file: "+err.Error())
		return 1
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		fmt.Fprintln(os.Stderr, "rename temp device file: "+err.Error())
		return 1
	}

	if err := config.ValidateFiles(absCfg); err != nil {
		// Rollback to previous content.
		rbTmp := filepath.Join(devicesDir, fmt.Sprintf(".%s.%d.rollback.tmp", *id, time.Now().UnixNano()))
		_ = os.WriteFile(rbTmp, oldBytes, 0o644)
		_ = os.Rename(rbTmp, path)
		fmt.Fprintln(os.Stderr, "config validation after update failed (rolled back): "+err.Error())
		return 1
	}

	fmt.Fprintln(os.Stdout, "device updated: "+path)
	return 0
}

func runDeviceRemove(args []string) int {
	fs := flag.NewFlagSet("device remove", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	id := fs.String("id", "", "Device ID (required)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	*id = strings.TrimSpace(*id)
	if *id == "" {
		fmt.Fprintln(os.Stderr, "id is required")
		return 2
	}

	absCfg, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	loaded, err := config.LoadFromFile(absCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(loaded); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	devicesDir := config.ResolvePath(filepath.Dir(absCfg), loaded.Root.Paths.DevicesDir)
	path, err := findDeviceFileByID(devicesDir, *id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "device not found: "+*id)
			return 1
		}
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	oldBytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := os.Remove(path); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.ValidateFiles(absCfg); err != nil {
		// Rollback deletion.
		_ = os.WriteFile(path, oldBytes, 0o644)
		fmt.Fprintln(os.Stderr, "config validation after remove failed (rolled back): "+err.Error())
		return 1
	}

	fmt.Fprintln(os.Stdout, "device removed: "+path)
	return 0
}

func runDeviceTest(args []string) int {
	fs := flag.NewFlagSet("device test", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	id := fs.String("id", "", "Device ID (required)")
	var snmp triBool
	var icmp triBool
	fs.Var(&snmp, "snmp", "Force SNMP enabled (true/false)")
	fs.Var(&icmp, "icmp", "Force ICMP enabled (true/false)")
	icmpCount := fs.Int("icmp-count", 2, "Ping count per target")
	icmpTimeout := fs.String("icmp-timeout", "2s", "Ping timeout per target (Go duration)")
	snmpCommunity := fs.String("snmp-community", "public", "SNMP community")
	snmpPort := fs.Int("snmp-port", 161, "SNMP port")
	snmpRetries := fs.Int("snmp-retries", 1, "SNMP retries")
	snmpTimeout := fs.String("snmp-timeout", "2s", "SNMP timeout (Go duration)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	*id = strings.TrimSpace(*id)
	if *id == "" {
		fmt.Fprintln(os.Stderr, "id is required")
		return 2
	}

	absCfg, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	loaded, err := config.LoadFromFile(absCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(loaded); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	var dev *config.Device
	for i := range loaded.Devices {
		if loaded.Devices[i].ID == *id {
			dev = &loaded.Devices[i]
			break
		}
	}
	if dev == nil {
		fmt.Fprintln(os.Stderr, "device not found: "+*id)
		return 1
	}

	doICMP := dev.ICMP.Enabled
	doSNMP := dev.SNMP.Enabled
	if icmp.set {
		doICMP = icmp.value
	}
	if snmp.set {
		doSNMP = snmp.value
	}
	if !doICMP && !doSNMP {
		fmt.Fprintln(os.Stderr, "both icmp and snmp are disabled")
		return 2
	}

	// Stable preface for logs.
	fmt.Fprintf(os.Stdout, "device=%s address=%s vendor=%s model=%s\n", dev.ID, dev.Address, dev.Vendor, dev.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if doICMP {
		to, err := time.ParseDuration(*icmpTimeout)
		if err != nil || to <= 0 {
			to = 2 * time.Second
		}
		c := collectors.ICMPCollector{
			Targets: []collectors.Target{{DeviceID: dev.ID, Address: dev.Address}},
			Count:   *icmpCount,
			Timeout: to,
		}
		samples, err := c.Collect(ctx)
		if err != nil {
			fmt.Fprintf(os.Stdout, "icmp status=error reason=%q\n", err.Error())
		} else {
			// Summarize.
			vals := map[string]float64{}
			for _, s := range samples {
				m, _ := s.Fields["metric"].(string)
				v, _ := s.Fields["value_number"].(float64)
				if m != "" {
					vals[m] = v
				}
			}
			fmt.Fprintf(os.Stdout, "icmp reachable=%v latency_ms=%.2f jitter_ms=%.2f loss_pct=%.0f\n",
				vals["icmp.reachable"],
				vals["icmp.latency_ms"],
				vals["icmp.jitter_ms"],
				vals["icmp.packet_loss_pct"],
			)
		}
	}

	if doSNMP {
		to, err := time.ParseDuration(*snmpTimeout)
		if err != nil || to <= 0 {
			to = 2 * time.Second
		}
		port := *snmpPort
		if port <= 0 || port > 65535 {
			fmt.Fprintln(os.Stderr, "invalid snmp-port")
			return 2
		}

		profilesDir := loaded.ProfilesDir
		profs, err := profiles.LoadDir(filepath.Clean(profilesDir))
		if err != nil {
			fmt.Fprintf(os.Stdout, "snmp status=error reason=%q\n", err.Error())
			return 1
		}
		if err := profiles.ValidateAll(profs); err != nil {
			fmt.Fprintf(os.Stdout, "snmp status=error reason=%q\n", err.Error())
			return 1
		}
		if p, ok := profiles.SelectProfile(profs, dev.Vendor, dev.Model); ok {
			fmt.Fprintf(os.Stdout, "snmp profile=%s\n", p.Name)
		}

		c := collectors.SNMPCollector{
			Targets:   []collectors.Target{{DeviceID: dev.ID, Address: dev.Address, Vendor: dev.Vendor, Model: dev.Model}},
			Profiles:  profs,
			Community: strings.TrimSpace(*snmpCommunity),
			Port:      uint16(port),
			Timeout:   to,
			Retries:   *snmpRetries,
		}
		samples, err := c.Collect(ctx)
		if err != nil {
			fmt.Fprintf(os.Stdout, "snmp status=error reason=%q\n", err.Error())
			return 1
		}
		if len(samples) == 0 {
			fmt.Fprintln(os.Stdout, "snmp status=error reason=\"no samples\"")
			return 1
		}
		fmt.Fprintf(os.Stdout, "snmp status=ok samples=%d\n", len(samples))
	}

	return 0
}
