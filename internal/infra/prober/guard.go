package prober

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"

	"github.com/St1lon/sentinel/internal/domain"
)

var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),      // "этот" хост
	netip.MustParsePrefix("10.0.0.0/8"),     // частная сеть
	netip.MustParsePrefix("100.64.0.0/10"),  // CGNAT
	netip.MustParsePrefix("127.0.0.0/8"),    // loopback
	netip.MustParsePrefix("169.254.0.0/16"), // link-local, включая облачные метаданные
	netip.MustParsePrefix("172.16.0.0/12"),  // частная сеть
	netip.MustParsePrefix("192.0.0.0/24"),   // служебные назначения IETF
	netip.MustParsePrefix("192.168.0.0/16"), // частная сеть
	netip.MustParsePrefix("198.18.0.0/15"),  // бенчмарки
	netip.MustParsePrefix("224.0.0.0/4"),    // multicast
	netip.MustParsePrefix("240.0.0.0/4"),    // зарезервировано
	netip.MustParsePrefix("::1/128"),        // IPv6 loopback
	netip.MustParsePrefix("::/128"),         // IPv6 unspecified
	netip.MustParsePrefix("fc00::/7"),       // IPv6 unique local
	netip.MustParsePrefix("fe80::/10"),      // IPv6 link-local
	netip.MustParsePrefix("ff00::/8"),       // IPv6 multicast
	netip.MustParsePrefix("64:ff9b::/96"),   // NAT64, может указывать на внутренний IPv4
	netip.MustParsePrefix("2002::/16"),      // 6to4
}

type Guard struct {
	allowPrivate bool
}

func NewGuard(allowPrivate bool) *Guard {
	return &Guard{allowPrivate: allowPrivate}
}

func (g *Guard) CheckURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidTarget, err.Error())
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%w: only http and https are supported", domain.ErrInvalidTarget)
	}

	if parsed.Host == "" {
		return nil, fmt.Errorf("%w: host is empty", domain.ErrInvalidTarget)
	}

	if parsed.User != nil {
		return nil, fmt.Errorf("%w: credentials in url are not allowed", domain.ErrInvalidTarget)
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("%w: host is empty", domain.ErrInvalidTarget)
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		if err := g.CheckAddr(addr); err != nil {
			return nil, err
		}
	}

	return parsed, nil
}

func (g *Guard) CheckAddr(addr netip.Addr) error {
	if g.allowPrivate {
		return nil
	}

	addr = addr.Unmap()

	if !addr.IsValid() {
		return fmt.Errorf("%w: invalid ip address", domain.ErrTargetNotAllowed)
	}

	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return fmt.Errorf("%w: %s is in reserved range %s", domain.ErrTargetNotAllowed, addr, prefix)
		}
	}

	return nil
}

// Вызывается перед connect с уже разрезолвленным адресом — закрывает DNS rebinding.
func (g *Guard) CheckDialAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: cannot parse dial address", domain.ErrTargetNotAllowed)
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("%w: cannot parse dial ip", domain.ErrTargetNotAllowed)
	}

	return g.CheckAddr(addr)
}
