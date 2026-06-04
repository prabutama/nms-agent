package discovery

import "strings"

func ResolveFingerprintVendorModel(fp Fingerprint) (string, string) {
	sysObjectID := strings.TrimSpace(fp.SysObjectID)
	sysDescr := strings.ToLower(strings.TrimSpace(fp.SysDescr))

	if strings.HasPrefix(sysObjectID, "1.3.6.1.4.1.14988") || strings.Contains(sysDescr, "mikrotik") || strings.Contains(sysDescr, "routeros") {
		return "mikrotik", "routeros"
	}

	if strings.Contains(sysDescr, "proxmox") {
		return "linux", "proxmox"
	}
	if strings.Contains(sysDescr, "ubuntu") {
		return "linux", "ubuntu"
	}
	if strings.Contains(sysDescr, "debian") {
		return "linux", "debian"
	}
	if strings.Contains(sysDescr, "linux") {
		return "linux", ""
	}

	return "", ""
}
