package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
)

type ChangeCache struct {
	mu    sync.Mutex
	state map[string]string
}

func NewChangeCache() *ChangeCache {
	return &ChangeCache{state: map[string]string{}}
}

func (c *ChangeCache) Update(deviceID, addressFamily, fingerprint string) bool {
	if c == nil {
		return true
	}
	key := deviceID + ":" + addressFamily
	c.mu.Lock()
	defer c.mu.Unlock()
	prev, ok := c.state[key]
	c.state[key] = fingerprint
	return ok && prev != fingerprint
}

func Fingerprint(routes []RouteEntry) string {
	if len(routes) == 0 {
		return ""
	}
	b := strings.Builder{}
	for _, route := range routes {
		b.WriteString(route.Destination)
		b.WriteByte('|')
		b.WriteString(route.NextHop)
		b.WriteByte('|')
		b.WriteString(route.InterfaceID)
		b.WriteByte('|')
		b.WriteString(route.Protocol)
		b.WriteByte('|')
		b.WriteString(route.RouteType)
		b.WriteByte('|')
		b.WriteString(strconv.Itoa(route.Metric))
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
