package collectors

// Target is a minimal device identity for collectors.
// It deliberately avoids importing config packages to keep collectors reusable.
type Target struct {
	DeviceID string
	Address  string
	Vendor   string
	Model    string
}
