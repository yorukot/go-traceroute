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
		Method:  traceroute.MethodICMP,
		MaxHops: 30,
		Timeout: 2 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	events, err := tr.TraceStream(ctx, "example.com")
	if err != nil {
		panic(err)
	}

	for event := range events {
		switch event.Kind {
		case traceroute.EventHop:
			fmt.Printf("hop %d\n", event.Hop.TTL)
		case traceroute.EventDone:
			fmt.Printf("done: %s\n", event.Trace.Destination)
		case traceroute.EventError:
			if errors.Is(event.Error, traceroute.ErrPermission) {
				fmt.Println("raw socket permission required")
				return
			}
			if errors.Is(event.Error, traceroute.ErrUnsupported) {
				fmt.Println("trace backend is not supported yet")
				return
			}
			panic(event.Error)
		}
	}
}
