// Package traceroute provides a library for tracing network paths using
// ICMP, UDP, or TCP probes.
//
// Most full traceroute modes require permission to create raw sockets. Callers
// should expect ErrPermission on systems where the current process cannot open
// the required socket.
//
// The package returns structured trace results. It does not format terminal
// output.
package traceroute
