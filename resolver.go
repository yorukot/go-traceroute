package traceroute

import (
	"context"
	"net"
	"net/netip"
)

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
	return names, nil
}
