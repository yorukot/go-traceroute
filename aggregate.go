package traceroute

func reachedDestination(hop Hop) bool {
	for _, probe := range hop.Probes {
		if probe.Status == StatusDestination {
			return true
		}
	}
	return false
}
