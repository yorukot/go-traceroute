package traceroute_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yorukot/go-traceroute"
)

func ExampleNew() {
	tr, err := traceroute.New(traceroute.Options{
		MaxHops:       64,
		QueriesPerHop: 3,
		Timeout:       2 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	_ = tr
}

func ExampleTraceRoute_permissionHandling() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := traceroute.TraceRoute(ctx, "127.0.0.1", traceroute.Options{
		MaxHops: 1,
	})
	if err != nil {
		if errors.Is(err, traceroute.ErrPermission) {
			return
		}
		panic(err)
	}

	for _, hop := range result.Hops {
		fmt.Println(hop.TTL)
	}
}
