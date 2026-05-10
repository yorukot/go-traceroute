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

	result, err := tr.Trace(ctx, "1.1.1.1")
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
