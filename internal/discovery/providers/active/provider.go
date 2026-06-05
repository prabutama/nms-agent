package active

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"time"

	"nms-agent/internal/config"
	"nms-agent/internal/discovery"
)

const (
	defaultProbeTimeout     = time.Second
	defaultProbeConcurrency = 64
)

type Provider struct {
	Ping func(ctx context.Context, address string, timeout time.Duration) bool
}

func (p Provider) Candidates(ctx context.Context, loaded config.Loaded) ([]discovery.Candidate, error) {
	iface, err := net.InterfaceByName(loaded.Root.Discovery.Interface)
	if err != nil {
		return nil, err
	}
	_, subnet, err := net.ParseCIDR(loaded.Root.Discovery.Subnet)
	if err != nil {
		return nil, err
	}
	if subnet.IP.To4() == nil {
		return nil, fmt.Errorf("active discovery currently supports IPv4 subnets only")
	}
	localIPs, err := interfaceIPv4s(iface)
	if err != nil {
		return nil, err
	}
	hosts := hostsInSubnet(subnet, localIPs)
	if len(hosts) == 0 {
		return nil, nil
	}
	timeout := loaded.Root.Discovery.ActiveProbe.Timeout
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	concurrency := loaded.Root.Discovery.ActiveProbe.Concurrency
	if concurrency <= 0 {
		concurrency = defaultProbeConcurrency
	}
	pingFn := p.Ping
	if pingFn == nil {
		pingFn = pingReachable
	}

	type result struct {
		addr string
		ok   bool
	}
	jobs := make(chan string)
	results := make(chan result, len(hosts))
	workers := concurrency
	if workers > len(hosts) {
		workers = len(hosts)
	}
	for i := 0; i < workers; i++ {
		go func() {
			for addr := range jobs {
				probeCtx, cancel := context.WithTimeout(ctx, timeout)
				ok := pingFn(probeCtx, addr, timeout)
				cancel()
				results <- result{addr: addr, ok: ok}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, host := range hosts {
			select {
			case <-ctx.Done():
				return
			case jobs <- host:
			}
		}
	}()

	out := make([]discovery.Candidate, 0, len(hosts))
	for i := 0; i < len(hosts); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case res := <-results:
			if !res.ok {
				continue
			}
			out = append(out, discovery.Candidate{
				Address:   res.addr,
				Interface: loaded.Root.Discovery.Interface,
				Source:    fmt.Sprintf("active:%s", loaded.Root.Discovery.Interface),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	return out, nil
}

func interfaceIPv4s(iface *net.Interface) (map[string]struct{}, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(addrs))
	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			if ip := v.IP.To4(); ip != nil {
				out[ip.String()] = struct{}{}
			}
		case *net.IPAddr:
			if ip := v.IP.To4(); ip != nil {
				out[ip.String()] = struct{}{}
			}
		}
	}
	return out, nil
}

func hostsInSubnet(subnet *net.IPNet, localIPs map[string]struct{}) []string {
	base := subnet.IP.Mask(subnet.Mask).To4()
	if base == nil {
		return nil
	}
	mask := net.IP(subnet.Mask).To4()
	broadcast := net.IPv4(base[0]|^mask[0], base[1]|^mask[1], base[2]|^mask[2], base[3]|^mask[3]).To4()
	start := ipv4ToUint32(base)
	end := ipv4ToUint32(broadcast)
	out := make([]string, 0, maxInt(0, int(end-start)-1))
	for v := start + 1; v < end; v++ {
		ip := uint32ToIPv4(v).String()
		if _, skip := localIPs[ip]; skip {
			continue
		}
		out = append(out, ip)
	}
	return out
}

func pingReachable(ctx context.Context, address string, timeout time.Duration) bool {
	cmd := exec.CommandContext(ctx, "ping", pingArgs(address, timeout)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	return cmd.Run() == nil
}

func pingArgs(address string, timeout time.Duration) []string {
	if runtime.GOOS == "windows" {
		ms := int(timeout.Milliseconds())
		if ms <= 0 {
			ms = 1000
		}
		return []string{"-n", "1", "-w", strconv.Itoa(ms), address}
	}
	sec := int(timeout.Seconds())
	if sec <= 0 {
		sec = 1
	}
	return []string{"-c", "1", "-W", strconv.Itoa(sec), address}
}

func ipv4ToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIPv4(v uint32) net.IP {
	return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
