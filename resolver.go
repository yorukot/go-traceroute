package traceroute

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
)

// Resolver allows callers to provide deterministic or custom DNS behavior.
type Resolver interface {
	LookupIP(ctx context.Context, host string, version IPVersion) ([]netip.Addr, error)
	LookupAddr(ctx context.Context, addr netip.Addr) ([]string, error)
}

type defaultResolver struct {
	resolver *net.Resolver
}

func (r defaultResolver) LookupIP(ctx context.Context, host string, version IPVersion) ([]netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		if addressMatchesVersion(addr, version) {
			return []netip.Addr{addr}, nil
		}
		return nil, ErrNoAddress
	}

	network := "ip"
	switch version {
	case IPv4:
		network = "ip4"
	case IPv6:
		network = "ip6"
	}

	resolver := r.resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	addrs, err := resolver.LookupNetIP(ctx, network, host)
	if err != nil {
		return nil, err
	}

	filtered := addrs[:0]
	for _, addr := range addrs {
		if addressMatchesVersion(addr, version) {
			filtered = append(filtered, addr)
		}
	}
	if len(filtered) == 0 {
		return nil, ErrNoAddress
	}

	return filtered, nil
}

func (r defaultResolver) LookupAddr(ctx context.Context, addr netip.Addr) ([]string, error) {
	resolver := r.resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	names, err := resolver.LookupAddr(ctx, addr.String())
	if err != nil {
		return nil, err
	}
	for i := range names {
		names[i] = strings.TrimSuffix(names[i], ".")
	}
	return names, nil
}

func addressMatchesVersion(addr netip.Addr, version IPVersion) bool {
	switch version {
	case IPAny:
		return true
	case IPv4:
		return addr.Is4()
	case IPv6:
		return addr.Is6()
	default:
		return false
	}
}

func normalizeResolverError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNoAddress) {
		return ErrNoAddress
	}
	return err
}
