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
