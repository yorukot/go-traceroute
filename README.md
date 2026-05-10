# go-traceroute

`go-traceroute` is a Go library for ICMP traceroute with structured results.

The public API is intentionally small: callers configure a `Tracer`, run a
blocking trace with `Trace`, or stream progress with `TraceStream`. Formatting
classic traceroute output is left to the caller.

```go
tr, err := traceroute.New(traceroute.Options{
    MaxHops:       64,
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

## Permissions

ICMP tracing uses raw sockets on most systems. Run with the required privileges
or capabilities for your platform, otherwise traces return `ErrPermission`.

## Examples

See `examples/basic`, `examples/stream`, and `examples/mtr`.

## License

MIT.
