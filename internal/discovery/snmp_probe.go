package discovery

import (
	"context"
	"os"
	"strings"
	"time"

	g "github.com/gosnmp/gosnmp"

	"nms-agent/internal/config"
)

const (
	oidSysDescr    = "1.3.6.1.2.1.1.1.0"
	oidSysObjectID = "1.3.6.1.2.1.1.2.0"
	oidSysName     = "1.3.6.1.2.1.1.5.0"
)

type SNMPProber struct{}

func (p SNMPProber) Probe(ctx context.Context, candidate Candidate, cfg config.DiscoverySNMP) (Fingerprint, error) {
	fp := Fingerprint{Candidate: candidate}

	ver := g.Version2c
	if strings.TrimSpace(cfg.Version) == "v2c" {
		ver = g.Version2c
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d > 0 && d < timeout {
			timeout = d
		}
	}
	retries := cfg.Retries
	if retries <= 0 {
		retries = 1
	}
	client := &g.GoSNMP{
		Target:    candidate.Address,
		Port:      161,
		Community: expandSNMPCommunity(cfg.Community),
		Version:   ver,
		Timeout:   timeout,
		Retries:   retries,
	}
	if err := client.Connect(); err != nil {
		return fp, nil
	}
	defer func() {
		if client.Conn != nil {
			_ = client.Conn.Close()
		}
	}()
	pkt, err := client.Get([]string{oidSysObjectID, oidSysName, oidSysDescr})
	if err != nil {
		return fp, nil
	}
	fp.SNMPOK = true
	for _, v := range pkt.Variables {
		switch v.Name {
		case oidSysObjectID:
			if s, ok := pduToString(v); ok {
				fp.SysObjectID = s
			}
		case oidSysName:
			if s, ok := pduToString(v); ok {
				fp.SysName = s
			}
		case oidSysDescr:
			if s, ok := pduToString(v); ok {
				fp.SysDescr = s
			}
		}
	}
	return fp, nil
}

func expandSNMPCommunity(s string) string {
	s = strings.TrimSpace(os.ExpandEnv(s))
	if s == "" {
		return "public"
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, "\r", ""), "\n", ""), "\t", ""))
}

func pduToString(pdu g.SnmpPDU) (string, bool) {
	switch v := pdu.Value.(type) {
	case string:
		return strings.TrimSpace(v), true
	case []byte:
		return strings.TrimSpace(string(v)), true
	default:
		return "", false
	}
}
