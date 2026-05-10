# go-traceroute

`go-traceroute` is a Go library for tracing network paths and returning
structured results.

The public API is intentionally small: callers configure a `Tracer`, run a
blocking trace with `Trace`, or stream progress with `TraceStream`. Formatting
classic traceroute output is left to the caller.

```go
tr, err := traceroute.New(traceroute.Options{
    Method:        traceroute.MethodICMP,
    MaxHops:       30,
    QueriesPerHop: 3,
    Timeout:       2 * time.Second,
})
if err != nil {
    return err
}

result, err := tr.Trace(ctx, "example.com")
if err != nil {
    return err
}

for _, hop := range result.Hops {
    fmt.Println(hop.TTL, hop.Probes)
}
```

## Status

This repository currently contains the public API, internal engine, probe
interfaces, fake test utilities, and examples. Real ICMP/UDP/TCP socket
backends are still pending, so traces currently return `ErrUnsupported`.

## Examples

See `examples/basic` and `examples/stream`.

## License

MIT.
