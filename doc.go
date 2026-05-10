// Package traceroute provides a small ICMP and UDP traceroute library.
//
// Most full traceroute modes require permission to create raw sockets. Callers
// should expect ErrPermission on systems where the current process cannot open
// the required socket.
//
// The package returns structured trace results. It does not format terminal
// output.
package traceroute
