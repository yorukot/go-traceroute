package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/yorukot/go-traceroute"
)

const defaultTarget = "1.1.1.1"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var maxRounds int
	flag.IntVar(&maxRounds, "count", 0, "maximum trace rounds to run; 0 means run until interrupted")
	flag.IntVar(&maxRounds, "c", 0, "maximum trace rounds to run; 0 means run until interrupted")
	flag.Parse()
	if maxRounds < 0 {
		fmt.Fprintln(os.Stderr, "count must be >= 0")
		os.Exit(2)
	}

	target := defaultTarget
	if flag.NArg() > 0 {
		target = flag.Arg(0)
	}

	tr, err := traceroute.New(traceroute.Options{
		MaxHops:       64,
		QueriesPerHop: 1,
		Timeout:       2 * time.Second,
		ResolveNames:  true,
	})
	if err != nil {
		panic(err)
	}

	stats := make(map[int]*hopStats)

	if maxRounds > 0 {
		fmt.Printf("mtr-style trace to %s for %d rounds. Press Ctrl-C to stop early.\n", target, maxRounds)
	} else {
		fmt.Printf("mtr-style trace to %s. Press Ctrl-C to stop.\n", target)
	}
	printHeader()

	for round := 1; ctx.Err() == nil && (maxRounds == 0 || round <= maxRounds); round++ {
		events, err := tr.TraceStream(ctx, target)
		if err != nil {
			panic(err)
		}

		for event := range events {
			switch event.Kind {
			case traceroute.EventHop:
				if event.Hop == nil {
					continue
				}
				stat := stats[event.Hop.TTL]
				if stat == nil {
					stat = &hopStats{}
					stats[event.Hop.TTL] = stat
				}
				stat.add(event.Hop.Probes)
				printHop(event.Hop.TTL, stat)
			case traceroute.EventDone:
				if event.Trace != nil {
					fmt.Printf("round %d complete: %s\n", round, event.Trace.Destination)
					printHeader()
				}
			case traceroute.EventError:
				if errors.Is(event.Error, context.Canceled) {
					return
				}
				if errors.Is(event.Error, traceroute.ErrPermission) {
					fmt.Println("raw socket permission required")
					return
				}
				panic(event.Error)
			}
		}

		if maxRounds > 0 && round >= maxRounds {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

type hopStats struct {
	host string

	sent     int
	received int

	last  time.Duration
	total time.Duration
	best  time.Duration
	worst time.Duration

	sumSquares float64
}

func (s *hopStats) add(probes []traceroute.Probe) {
	for _, probe := range probes {
		s.sent++

		if probe.Hostname != "" {
			s.host = probe.Hostname
		} else if probe.Addr.IsValid() {
			s.host = probe.Addr.String()
		}

		if probe.Status == traceroute.StatusTimeout || probe.Status == traceroute.StatusError {
			continue
		}
		if !probe.Addr.IsValid() {
			continue
		}

		s.received++
		s.last = probe.RTT
		s.total += probe.RTT

		if s.best == 0 || probe.RTT < s.best {
			s.best = probe.RTT
		}
		if probe.RTT > s.worst {
			s.worst = probe.RTT
		}

		ms := durationMillis(probe.RTT)
		s.sumSquares += ms * ms
	}
}

func printHeader() {
	fmt.Printf("%3s  %-36s %6s %5s %7s %7s %7s %7s %7s\n",
		"", "Host", "Loss%", "Snt", "Last", "Avg", "Best", "Wrst", "StDev")
}

func printHop(ttl int, stat *hopStats) {
	host := stat.host
	if host == "" {
		host = "(waiting for reply)"
	}

	fmt.Printf("%3d. %-36s %6.1f %5d %7s %7s %7s %7s %7s\n",
		ttl,
		truncate(host, 36),
		stat.lossPercent(),
		stat.sent,
		formatDuration(stat.last, stat.received),
		formatDuration(stat.avg(), stat.received),
		formatDuration(stat.best, stat.received),
		formatDuration(stat.worst, stat.received),
		formatFloat(stat.stddev(), stat.received),
	)
}

func (s *hopStats) lossPercent() float64 {
	if s.sent == 0 {
		return 0
	}
	return float64(s.sent-s.received) * 100 / float64(s.sent)
}

func (s *hopStats) avg() time.Duration {
	if s.received == 0 {
		return 0
	}
	return s.total / time.Duration(s.received)
}

func (s *hopStats) stddev() float64 {
	if s.received == 0 {
		return 0
	}

	mean := durationMillis(s.avg())
	variance := s.sumSquares/float64(s.received) - mean*mean
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance)
}

func formatDuration(d time.Duration, received int) string {
	if received == 0 {
		return "*"
	}
	return fmt.Sprintf("%.1f", durationMillis(d))
}

func formatFloat(v float64, received int) string {
	if received == 0 {
		return "*"
	}
	return fmt.Sprintf("%.1f", v)
}

func durationMillis(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "."
}
