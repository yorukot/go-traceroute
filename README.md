# go-traceroute

[![Go Report Card](https://goreportcard.com/badge/github.com/yorukot/go-traceroute)](https://goreportcard.com/report/github.com/yorukot/go-traceroute)

`go-traceroute` is a Go library for ICMP and UDP (maybe more in the future) traceroute with structured results.

The public API is intentionally small: callers configure a `Tracer`, run a blocking trace with `Trace`, or stream progress with `TraceStream`. Formatting of classic traceroute output is left to the caller.

## Installation

```sh
go get github.com/yorukot/go-traceroute
```

## Usage

For runnable code, see the examples:

- `examples/basic`: blocking `Trace` usage.
- `examples/stream`: streaming `TraceStream` events.
- `examples/mtr`: repeated traces with MTR-style stats.
- `examples/udp`: UDP traceroute usage.

Use `Trace` when you want to run a complete trace and inspect the final structured result. Use `TraceStream` when you want progress events while the trace is still running. The stream emits probe, hop, done, and error events.

By default, `IPVersion` is `IPAny`, so the resolver may return either IPv4 or IPv6. Set it to `IPv4` or `IPv6` when you need one family.

ICMP is the default probe protocol. Select `ProtocolUDP` when you want classic UDP traceroute behavior.

UDP probes use `UDPBasePort` plus the probe offset for the destination port, so each TTL and attempt can be matched to its ICMP response.

## Options

`Options` controls how probes are sent:

- `Protocol`: probe protocol. Use `ProtocolICMP` or `ProtocolUDP`. Default is `ProtocolICMP`.
- `IPVersion`: address family selection. Use `IPAny`, `IPv4`, or `IPv6`.
- `FirstHop`: first TTL or IPv6 Hop Limit to probe. Default is `1`.
- `MaxHops`: maximum TTL or Hop Limit to probe. Default is `64`.
- `QueriesPerHop`: number of probes sent for each hop. Default is `3`.
- `Timeout`: timeout for each probe response. Default is `3s`.
- `PacketSize`: probe packet size in bytes. Default is `48`.
- `UDPBasePort`: first UDP destination port when using `ProtocolUDP`. Default is `33434`.
- `ResolveNames`: enable reverse DNS lookup for responding hop addresses.

## Results

`Trace` returns a `Trace` with the resolved destination, IP version, timestamps,
and one `Hop` per TTL or Hop Limit. Each `Probe` includes the responding
address, optional hostname, RTT, status, ICMP type/code, and optional
annotation.

Common probe statuses are:

- `StatusOK`: an intermediate hop replied.
- `StatusDestination`: the destination replied.
- `StatusTimeout`: no matching reply was received before the probe timeout.
- `StatusUnreachable`: an ICMP destination unreachable response was received.
- `StatusFiltered`: the path appears to be administratively filtered.
- `StatusPacketTooBig`: an ICMP packet-too-big response was received.
- `StatusError`: sending or receiving the probe failed.

## Errors

The package exposes sentinel errors for common cases:

- `ErrPermission`: raw socket permission is missing.
- `ErrNoAddress`: the target did not resolve to a usable address for the selected `IPVersion`.
- `ErrTimeout`: the operation timed out.

Other errors may come from DNS, context cancellation, or socket operations.

## Permissions

ICMP tracing and UDP reply collection use raw ICMP sockets on most systems. Run
with the required privileges or capabilities for your platform, otherwise traces
return `ErrPermission`.

## Examples

See `examples/basic`, `examples/udp`, `examples/stream`, and `examples/mtr`.

## License

MIT.

### Star History

<a href="https://star-history.com/#yorukot/go-traceroute&Timeline">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=yorukot/go-traceroute&type=Timeline&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=yorukot/go-traceroute&type=Timeline" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=yorukot/go-traceroute&type=Timeline" />
 </picture>
</a>

<div align="center">

## ༼ つ ◕_◕ ༽つ Please share.

</div>
