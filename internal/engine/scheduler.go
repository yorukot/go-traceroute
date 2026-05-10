package engine

import (
	"context"
	"errors"
	"net/netip"

	"github.com/yorukot/go-traceroute/internal/probe"
)

func (e *Engine) traceHop(ctx context.Context, prober probe.Prober, ttl int) (Hop, error) {
	hop := Hop{TTL: ttl}

	for attempt := 1; attempt <= e.opts.QueriesPerHop; attempt++ {
		sent, err := prober.Send(ctx, ttl, attempt)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return hop, ctxErr
			}
			completed := Probe{
				Attempt: attempt,
				Status:  StatusError,
				Error:   err.Error(),
			}
			hop.Probes = append(hop.Probes, completed)
			e.emit(Event{Kind: EventProbe, Probe: &completed})
			continue
		}

		reply, err := prober.Receive(ctx, sent, e.opts.Timeout)
		completed := Probe{
			Attempt: attempt,
			SentAt:  sent.SentAt,
		}

		switch {
		case err == nil:
			completed = probeFromReply(attempt, sent, reply)
			if e.opts.ResolveNames && reply.From.IsValid() {
				completed.Hostname = e.lookupHostname(ctx, reply.From)
			}
		case errors.Is(err, probe.ErrTimeout):
			completed.Status = StatusTimeout
		case ctx.Err() != nil:
			return hop, ctx.Err()
		default:
			completed.Status = StatusError
			completed.Error = err.Error()
		}

		hop.Probes = append(hop.Probes, completed)
		e.emit(Event{Kind: EventProbe, Probe: &completed})
	}

	return hop, nil
}

func probeFromReply(attempt int, sent probe.Sent, reply probe.Reply) Probe {
	var status Status
	switch reply.Kind {
	case probe.ReplyTimeExceeded:
		status = StatusOK
	case probe.ReplyDestination:
		status = StatusDestination
	case probe.ReplyUnreachable:
		status = StatusUnreachable
	case probe.ReplyPacketTooBig:
		status = StatusPacketTooBig
	case probe.ReplyFiltered:
		status = StatusFiltered
	default:
		status = StatusError
	}

	receivedAt := reply.ReceivedAt
	rtt := reply.RTT
	if !receivedAt.IsZero() && rtt == 0 {
		rtt = receivedAt.Sub(sent.SentAt)
	}

	return Probe{
		Attempt:    attempt,
		Addr:       reply.From,
		SentAt:     sent.SentAt,
		ReceivedAt: receivedAt,
		RTT:        rtt,
		Status:     status,
		Annotation: reply.Annotation,
		ICMPType:   reply.ICMPType,
		ICMPCode:   reply.ICMPCode,
	}
}

func (e *Engine) lookupHostname(ctx context.Context, addr netip.Addr) string {
	names, err := e.res.LookupAddr(ctx, addr)
	if err != nil || len(names) == 0 {
		return ""
	}
	return names[0]
}
