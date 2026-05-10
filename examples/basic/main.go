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
		Method:        traceroute.MethodICMP,
		MaxHops:       30,
		QueriesPerHop: 3,
		Timeout:       2 * time.Second,
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
		if errors.Is(err, traceroute.ErrUnsupported) {
			fmt.Println("trace backend is not supported yet")
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
			fmt.Printf(" %s %.3fms", probe.Addr, float64(probe.RTT.Microseconds())/1000)
		}
		fmt.Println()
	}
}
