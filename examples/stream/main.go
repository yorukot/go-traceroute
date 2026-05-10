package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yorukot/go-traceroute"
)

const target = "1.1.1.1"

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

	events, err := tr.TraceStream(ctx, target)
	if err != nil {
		panic(err)
	}

	for event := range events {
		switch event.Kind {
		case traceroute.EventHop:
			printHop(event.Hop)
		case traceroute.EventDone:
			if event.Trace != nil {
				fmt.Printf("done: %s\n", event.Trace.Destination)
			}
		case traceroute.EventError:
			if errors.Is(event.Error, traceroute.ErrPermission) {
				fmt.Println("raw socket permission required")
				return
			}
			panic(event.Error)
		}
	}
}

func printHop(hop *traceroute.Hop) {
	if hop == nil {
		return
	}

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
