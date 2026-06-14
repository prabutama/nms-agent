package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nms-agent/internal/config"
	"nms-agent/internal/discovery"
	discoveryexplorer "nms-agent/internal/discovery/explorer"
	discoveryproviders "nms-agent/internal/discovery/providers"
)

type udpDialerFunc func(network, address string) (net.Conn, error)

var defaultInterfaceResolver = interfaceResolver{
	dial:       net.Dial,
	interfaces: net.Interfaces,
}

type interfaceResolver struct {
	dial       udpDialerFunc
	interfaces func() ([]net.Interface, error)
}

var newDiscoveryService = func(loaded config.Loaded) discovery.Service {
	return discovery.Service{
		Provider: discoveryproviders.New(loaded),
		Prober:   discovery.SNMPProber{},
		Explorer: discoveryexplorer.Explorer{},
	}
}

func runDiscovery(args []string) int {
	if len(args) < 1 {
		discoveryUsage()
		return 2
	}
	if isHelpArg(args[0]) {
		discoveryUsage()
		return 0
	}
	switch args[0] {
	case "run":
		return runDiscoveryRun(args[1:])
	case "preview":
		return runDiscoveryPreview(args[1:])
	case "status":
		return runDiscoveryStatus(args[1:])
	default:
		discoveryUsage()
		return 2
	}
}

func discoveryUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  nms-agentctl discovery status")
	fmt.Fprintln(os.Stderr, "  nms-agentctl discovery preview --subnet <cidr> [--interface <name>] [--max-new-devices 50]")
	fmt.Fprintln(os.Stderr, "  nms-agentctl discovery run --subnet <cidr> [--interface <name>] [--max-new-devices 50]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Config default: /etc/nms-agent/agent.yml")
	fmt.Fprintln(os.Stderr, "Override with: --config <path>")
}

func runDiscoveryRun(args []string) int {
	return runDiscoveryCommand(args, true)
}

func runDiscoveryPreview(args []string) int {
	return runDiscoveryCommand(args, false)
}

func runDiscoveryCommand(args []string, apply bool) int {
	name := "discovery preview"
	if apply {
		name = "discovery run"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	subnet := fs.String("subnet", "", "Target discovery subnet in CIDR form")
	iface := fs.String("interface", "", "Local interface name (optional; auto-detect when omitted)")
	provider := fs.String("provider", "", "Discovery provider: netlink or active")
	community := fs.String("snmp-community", "", "SNMP community override (env vars expanded)")
	communityAlias := fs.String("community", "", "Deprecated alias for --snmp-community")
	timeout := fs.Duration("timeout", 0, "SNMP timeout override")
	retries := fs.Int("retries", -1, "SNMP retries override")
	concurrency := fs.Int("concurrency", 0, "SNMP concurrency override")
	maxNewDevices := fs.Int("max-new-devices", 0, "Max devices to promote: 0=default 50, >0=limit, -1=unlimited")
	writeTo := fs.String("write-to", "", "Device output directory override")
	explore := fs.Bool("explore", false, "Enable profile exploration for unknown devices")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*subnet) == "" {
		fmt.Fprintln(os.Stderr, "--subnet is required")
		return 2
	}
	configAbs, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	loaded, err := config.LoadFromFile(configAbs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(loaded); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	communityValue := *community
	if strings.TrimSpace(communityValue) == "" {
		communityValue = *communityAlias
	}
	if err := applyDiscoveryCLIOverrides(&loaded, *subnet, *iface, *provider, communityValue, *timeout, *retries, *concurrency, *maxNewDevices, *writeTo, *explore, apply); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if strings.TrimSpace(os.ExpandEnv(loaded.Root.Discovery.SNMP.Community)) == "" {
		fmt.Fprintln(os.Stderr, "warning: SNMP community is not set; using default public community for discovery")
	}
	svc := newDiscoveryService(loaded)
	var res discovery.Result
	if apply {
		res, err = svc.RunOnce(context.Background(), configAbs, loaded)
	} else {
		res, err = svc.PreviewOnce(context.Background(), configAbs, loaded)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	printDiscoveryResult(res, apply)
	return 0
}

func runDiscoveryStatus(args []string) int {
	fs := flag.NewFlagSet("discovery status", flag.ContinueOnError)
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
	d := loaded.Root.Discovery
	if !d.Enabled && strings.TrimSpace(d.Subnet) == "" {
		fmt.Fprintln(os.Stdout, "configured=false")
		return 0
	}
	fmt.Fprintf(os.Stdout, "configured=true\n")
	fmt.Fprintf(os.Stdout, "interface=%s\n", d.Interface)
	fmt.Fprintf(os.Stdout, "subnet=%s\n", d.Subnet)
	fmt.Fprintf(os.Stdout, "provider=%s\n", d.Provider)
	fmt.Fprintf(os.Stdout, "snmp.version=%s\n", d.SNMP.Version)
	fmt.Fprintf(os.Stdout, "snmp.timeout=%s\n", d.SNMP.Timeout)
	fmt.Fprintf(os.Stdout, "snmp.retries=%d\n", d.SNMP.Retries)
	fmt.Fprintf(os.Stdout, "snmp.concurrency=%d\n", d.SNMP.Concurrency)
	fmt.Fprintf(os.Stdout, "auto_promote.enabled=%t\n", d.AutoPromote.Enabled)
	fmt.Fprintf(os.Stdout, "auto_promote.require_profile_match=%t\n", d.AutoPromote.RequireProfileMatch)
	fmt.Fprintf(os.Stdout, "auto_promote.max_new_devices_per_cycle=%d\n", d.AutoPromote.MaxNewDevicesPerCycle)
	fmt.Fprintf(os.Stdout, "auto_promote.device_id_template=%s\n", d.AutoPromote.DeviceIDTemplate)
	fmt.Fprintf(os.Stdout, "exploration.enabled=%t\n", d.Exploration.Enabled)
	fmt.Fprintf(os.Stdout, "exploration.run_when=%s\n", d.Exploration.RunWhen)
	fmt.Fprintf(os.Stdout, "exploration.max_oids_per_device=%d\n", d.Exploration.MaxOIDsPerDevice)
	fmt.Fprintf(os.Stdout, "exploration.output_dir=%s\n", d.Exploration.OutputDir)
	return 0
}

func printDiscoveryResult(res discovery.Result, apply bool) {
	mode := "preview"
	if apply {
		mode = "run"
	}
	fmt.Fprintf(os.Stdout, "mode=%s\n", mode)
	fmt.Fprintf(os.Stdout, "candidates=%d\n", res.CandidatesFound)
	fmt.Fprintf(os.Stdout, "existing_skipped=%d\n", res.ExistingSkipped)
	fmt.Fprintf(os.Stdout, "snmp_ok=%d\n", res.SNMPOK)
	fmt.Fprintf(os.Stdout, "profile_matched=%d\n", res.ProfileMatched)
	fmt.Fprintf(os.Stdout, "generated_profiles=%d\n", res.GeneratedProfiles)
	fmt.Fprintf(os.Stdout, "promoted=%d\n", res.Promoted)
	fmt.Fprintf(os.Stdout, "changed=%t\n", res.Changed)
	if len(res.SkippedReasons) > 0 {
		fmt.Fprintln(os.Stdout, "skipped_reasons:")
		for _, reason := range res.SkippedReasons {
			fmt.Fprintf(os.Stdout, "- %s\n", reason)
		}
	}
}

func applyDiscoveryCLIOverrides(loaded *config.Loaded, subnet, iface, provider, community string, timeout time.Duration, retries, concurrency, maxNewDevices int, writeTo string, explore, apply bool) error {
	if loaded == nil {
		return errors.New("loaded config is required")
	}
	if _, ipnet, err := net.ParseCIDR(strings.TrimSpace(subnet)); err != nil || ipnet == nil {
		return errors.New("--subnet must be a valid CIDR")
	}
	resolvedIface := strings.TrimSpace(iface)
	if resolvedIface == "" {
		var err error
		resolvedIface, err = autoDetectInterface(subnet)
		if err != nil {
			return err
		}
	}
	d := loaded.Root.Discovery
	d.Enabled = true
	d.Subnet = strings.TrimSpace(subnet)
	d.Interface = resolvedIface
	if strings.TrimSpace(provider) != "" {
		d.Provider = strings.TrimSpace(provider)
	}
	if strings.TrimSpace(d.Provider) == "" {
		d.Provider = "active"
	}
	switch strings.TrimSpace(d.Provider) {
	case "netlink", "active":
		// ok
	default:
		return errors.New("--provider must be 'netlink' or 'active'")
	}
	if d.ActiveProbe.Timeout <= 0 {
		d.ActiveProbe.Timeout = time.Second
	}
	if d.ActiveProbe.Concurrency <= 0 {
		d.ActiveProbe.Concurrency = 64
	}
	if strings.TrimSpace(d.SNMP.Version) == "" {
		d.SNMP.Version = "v2c"
	}
	if strings.TrimSpace(community) != "" {
		d.SNMP.Community = os.ExpandEnv(community)
	}
	if d.SNMP.Timeout <= 0 {
		d.SNMP.Timeout = 2 * time.Second
	}
	if timeout > 0 {
		d.SNMP.Timeout = timeout
	}
	if d.SNMP.Retries <= 0 {
		d.SNMP.Retries = 1
	}
	if retries >= 0 {
		d.SNMP.Retries = retries
	}
	if d.SNMP.Concurrency <= 0 {
		d.SNMP.Concurrency = 4
	}
	if concurrency > 0 {
		d.SNMP.Concurrency = concurrency
	}
	if d.AutoPromote.DeviceIDTemplate == "" {
		d.AutoPromote.DeviceIDTemplate = "{{vendor}}-{{sys_name}}"
	}
	if strings.TrimSpace(writeTo) != "" {
		d.AutoPromote.WriteTo = strings.TrimSpace(writeTo)
	}
	if strings.TrimSpace(d.AutoPromote.WriteTo) == "" {
		d.AutoPromote.WriteTo = loaded.Root.Paths.DevicesDir
	}
	d.AutoPromote.Enabled = apply
	d.AutoPromote.RequireSNMPOK = true
	d.AutoPromote.RequireSysObjectID = true
	d.AutoPromote.RequireProfileMatch = true
	if maxNewDevices != 0 {
		d.AutoPromote.MaxNewDevicesPerCycle = maxNewDevices
	}
	d.AutoPromote.MaxNewDevicesPerCycle = discovery.NormalizeMaxNewDevicesLimit(d.AutoPromote.MaxNewDevicesPerCycle)
	d.Exploration.Enabled = explore
	if d.Exploration.Enabled && strings.TrimSpace(d.Exploration.RunWhen) == "" {
		d.Exploration.RunWhen = "no_profile_match"
	}
	loaded.Root.Discovery = d
	return config.Validate(*loaded)
}

func autoDetectInterface(subnet string) (string, error) {
	return defaultInterfaceResolver.autoDetectInterface(subnet)
}

func (r interfaceResolver) autoDetectInterface(subnet string) (string, error) {
	target, err := firstUsableHost(subnet)
	if err != nil {
		return "", err
	}
	dial := r.dial
	if dial == nil {
		dial = net.Dial
	}
	conn, err := dial("udp", net.JoinHostPort(target, "9"))
	if err != nil {
		return "", fmt.Errorf("cannot auto-detect interface; pass --interface explicitly: %w", err)
	}
	defer conn.Close()
	udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || udpAddr.IP == nil {
		return "", errors.New("cannot auto-detect interface; pass --interface explicitly")
	}
	interfaces := r.interfaces
	if interfaces == nil {
		interfaces = net.Interfaces
	}
	ifaces, err := interfaces()
	if err != nil {
		return "", fmt.Errorf("cannot auto-detect interface; pass --interface explicitly: %w", err)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.Equal(udpAddr.IP) {
					return iface.Name, nil
				}
			case *net.IPAddr:
				if v.IP.Equal(udpAddr.IP) {
					return iface.Name, nil
				}
			}
		}
	}
	return "", errors.New("cannot auto-detect interface; pass --interface explicitly")
}

func firstUsableHost(subnet string) (string, error) {
	_, ipnet, err := net.ParseCIDR(strings.TrimSpace(subnet))
	if err != nil {
		return "", errors.New("--subnet must be a valid CIDR")
	}
	ip := ipnet.IP.To4()
	if ip == nil {
		return "", errors.New("--subnet must be an IPv4 CIDR")
	}
	ones, bits := ipnet.Mask.Size()
	if bits != 32 || ones >= 31 {
		return "", errors.New("--subnet has no usable host")
	}
	host := append(net.IP(nil), ip...)
	host[3]++
	if !ipnet.Contains(host) {
		return "", errors.New("--subnet has no usable host")
	}
	return host.String(), nil
}
