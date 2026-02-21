package main

import "hash/fnv"

// eligible returns true if this client should receive the given release.
// Rules use OR logic: the first matching condition grants eligibility.
// If rollout is nil, the release is served to all clients unconditionally.
func eligible(rollout *Rollout, customer, region, machineID, version string) bool {
	if rollout == nil {
		return true
	}

	// Customer whitelist
	for _, c := range rollout.Customers {
		if c == customer {
			return true
		}
	}

	// Region whitelist
	for _, r := range rollout.Regions {
		if r == region {
			return true
		}
	}

	// Deterministic percentage rollout.
	// We hash (machineID + ":" + version) so that the same machine always lands
	// in the same bucket for a given version, but different versions produce
	// independent buckets (preventing "always first" or "always last" machines).
	if rollout.Percentage != nil && *rollout.Percentage > 0 {
		h := fnv.New32a()
		h.Write([]byte(machineID + ":" + version))
		bucket := int(h.Sum32() % 100)
		if bucket < *rollout.Percentage {
			return true
		}
	}

	return false
}
