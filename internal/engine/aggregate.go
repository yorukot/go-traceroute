package engine

func reachedDestination(hop Hop) bool {
	for _, probe := range hop.Probes {
		if probe.Status == StatusDestination {
			return true
		}
	}
	return false
}

func hopStatus(probes []Probe) Status {
	if len(probes) == 0 {
		return StatusTimeout
	}

	allTimeout := true
	for _, probe := range probes {
		if probe.Status == StatusDestination {
			return StatusDestination
		}
		if probe.Status != StatusTimeout {
			allTimeout = false
		}
	}
	if allTimeout {
		return StatusTimeout
	}

	for _, probe := range probes {
		if probe.Status != StatusTimeout {
			return probe.Status
		}
	}

	return StatusTimeout
}

func shouldStopAfter(hop Hop) bool {
	return reachedDestination(hop)
}
