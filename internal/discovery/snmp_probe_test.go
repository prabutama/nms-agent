package discovery

import (
	"testing"

	g "github.com/gosnmp/gosnmp"
)

func TestPduToOIDString_ObjectIdentifierString(t *testing.T) {
	pdu := g.SnmpPDU{Type: g.ObjectIdentifier, Value: "iso.3.6.1.4.1.14988.1"}
	got, ok := pduToOIDString(pdu)
	if !ok {
		t.Fatalf("expected oid string to parse")
	}
	if got != "1.3.6.1.4.1.14988.1" {
		t.Fatalf("got %q want %q", got, "1.3.6.1.4.1.14988.1")
	}
}

func TestNormalizeOIDString_StripsLeadingDot(t *testing.T) {
	got := normalizeOIDString(".1.3.6.1.4.1.14988.1")
	if got != "1.3.6.1.4.1.14988.1" {
		t.Fatalf("got %q want %q", got, "1.3.6.1.4.1.14988.1")
	}
}

func TestAssignFingerprintField_NormalizesResponseOIDName(t *testing.T) {
	var fp Fingerprint
	assignFingerprintField(&fp, g.SnmpPDU{
		Name:  ".1.3.6.1.2.1.1.2.0",
		Type:  g.ObjectIdentifier,
		Value: "iso.3.6.1.4.1.14988.1",
	})
	if fp.SysObjectID != "1.3.6.1.4.1.14988.1" {
		t.Fatalf("got %q want %q", fp.SysObjectID, "1.3.6.1.4.1.14988.1")
	}

	assignFingerprintField(&fp, g.SnmpPDU{Name: ".1.3.6.1.2.1.1.5.0", Value: []byte("router-a")})
	assignFingerprintField(&fp, g.SnmpPDU{Name: ".1.3.6.1.2.1.1.1.0", Value: []byte("MikroTik RouterOS")})
	if fp.SysName != "router-a" {
		t.Fatalf("got sysName %q want %q", fp.SysName, "router-a")
	}
	if fp.SysDescr != "MikroTik RouterOS" {
		t.Fatalf("got sysDescr %q want %q", fp.SysDescr, "MikroTik RouterOS")
	}
}
