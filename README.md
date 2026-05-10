# go-traceroute

[![Go Report Card](https://goreportcard.com/badge/github.com/yorukot/go-traceroute)](https://goreportcard.com/report/github.com/yorukot/go-traceroute)

`go-traceroute` is a Go library for ICMP traceroute with structured results.

The public API is intentionally small: callers configure a `Tracer`, run ablocking trace with `Trace`, or stream progress with `TraceStream`. Formatting classic traceroute output is left to the caller.

## Installation

```sh
go get github.com/yorukot/go-traceroute
```

## Usage

### Blocking trace

Use `Trace` when you want to run a complete trace and inspect the final
structured result.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/yorukot/go-traceroute"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    tr, err := traceroute.New(traceroute.Options{
        MaxHops:       64,
        QueriesPerHop: 3,
        Timeout:       2 * time.Second,
        ResolveNames:  true,
    })
    if err != nil {
        panic(err)
    }

    result, err := tr.Trace(ctx, "example.com")
    if err != nil {
        if errors.Is(err, traceroute.ErrPermission) {
            fmt.Println("raw socket permission required")
            return
        }
        panic(err)
    }

    for _, hop := range result.Hops {
        fmt.Printf("%d", hop.TTL)
        for _, probe := range hop.Probes {
            if probe.Status == traceroute.StatusTimeout {
                fmt.Print(" *")
                continue
            }

            host := probe.Addr.String()
            if probe.Hostname != "" {
                host = probe.Hostname
            }
            fmt.Printf(" %s %.3fms", host, float64(probe.RTT.Microseconds())/1000)
        }
        fmt.Println()
    }
}
```

### Streaming trace

Use `TraceStream` when you want progress events while the trace is still
running. The channel emits probe, hop, done, and error events.

```go
events, err := tr.TraceStream(ctx, "example.com")
if err != nil {
    return err
}

for event := range events {
    switch event.Kind {
    case traceroute.EventProbe:
        fmt.Printf("probe: %+v\n", event.Probe)
    case traceroute.EventHop:
        fmt.Printf("hop: %+v\n", event.Hop)
    case traceroute.EventDone:
        fmt.Printf("done: %s\n", event.Trace.Destination)
    case traceroute.EventError:
        return event.Error
    }
}
```

### IPv4 and IPv6

By default, `IPVersion` is `IPAny`, so the resolver may return either address
family. Set it explicitly when you need one family.

```go
tr, err := traceroute.New(traceroute.Options{
    IPVersion: traceroute.IPv6,
})
if err != nil {
    return err
}

result, err := tr.Trace(ctx, "2606:4700:4700::1111")
if err != nil {
    return err
}
fmt.Println(result.Destination, result.IPVersion)
```

For IPv4-only tracing, use `traceroute.IPv4`.

### Options

`Options` controls how probes are sent:

- `IPVersion`: address family selection. Use `IPAny`, `IPv4`, or `IPv6`.
- `FirstHop`: first TTL or IPv6 Hop Limit to probe. Default is `1`.
- `MaxHops`: maximum TTL or Hop Limit to probe. Default is `64`.
- `QueriesPerHop`: number of probes sent for each hop. Default is `3`.
- `Timeout`: timeout for each probe response. Default is `3s`.
- `PacketSize`: ICMP packet size in bytes. Default is `48`.
- `ResolveNames`: enables reverse DNS lookup for responding hop addresses.

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

### Errors

The package exposes sentinel errors for common cases:

```go
result, err := tr.Trace(ctx, "example.com")
switch {
case err == nil:
    _ = result
case errors.Is(err, traceroute.ErrPermission):
    // Raw socket permission is missing.
case errors.Is(err, traceroute.ErrNoAddress):
    // The target did not resolve to a usable address for the selected IPVersion.
case errors.Is(err, traceroute.ErrTimeout):
    // The operation timed out.
default:
    // DNS, context cancellation, or another socket error.
}
```

## Permissions

ICMP tracing uses raw sockets on most systems. Run with the required privileges or capabilities for your platform, otherwise traces return `ErrPermission`.

## Examples

See `examples/basic`, `examples/stream`, and `examples/mtr`.

## Roadmap

- Add more examples for IPv6, JSON output, and web service integration.
- Explore additional probe backends such as UDP and TCP.
- Add integration testing guidance for environments that allow raw sockets.

## License

MIT.
