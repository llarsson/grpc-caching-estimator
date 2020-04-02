package server

import (
	"log"
	"math"
	"time"
)

// This implementation embodies (our understanding of) Lee et al.
// "An Update-Risk Based Approach to TTL Estimation in Web Caching", 2002.
// https://doi.org/10.1109/WISE.2002.1181640
type updateRiskBasedStrategy struct {
	K   int
	rho float64
}

// compile-time check that we adhere to interface
var _ estimationStrategy = (*updateRiskBasedStrategy)(nil)

func (strat *updateRiskBasedStrategy) initialize() {
	if strat.K == 0 {
		strat.K = 2 // optimum from the paper
	}
	log.Printf("Using Update-Risk Based strategy (K=%d)", strat.K)
}

// This comes in no way from the original paper, but our interface demands it,
// so this should be a reasonable implementation of interval determination.
func (strat *updateRiskBasedStrategy) determineInterval(intervals *[]interval, verifications *[]verification, estimations *[]estimation) (time.Duration, error) {
	estimate, err := lastEstimation(estimations)
	if err != nil {
		log.Printf("No previous estimations, relying on default interval")
		return defaultInterval, nil
	}

	bounded := math.Max(estimate.validity.Seconds()/2.0, defaultInterval.Seconds())

	return time.Duration(bounded) * time.Second, nil
}

func (strat *updateRiskBasedStrategy) determineEstimation(intervals *[]interval, verifications *[]verification, estimations *[]estimation, _ time.Duration) (time.Duration, error) {
	mu := strat.averageUpdateFrequency(verifications)
	t := -1.0 / mu * math.Log(1.0-strat.rho)
	return time.Duration(t) * time.Second, nil
}

func (strat *updateRiskBasedStrategy) averageUpdateFrequency(verifications *[]verification) float64 {
	timestamps, updates := backwardsUpdateDistance(verifications, strat.K)
	if updates == 0 {
		log.Printf("No observed value updates yet, using 1.0 as update frequency")
		return 1.0
	}

	// We requested K updates back, but perhaps got less. So we must rely
	// on what we actually got back from the data.
	timespan := time.Now().Sub(timestamps[updates-1])

	return float64(updates) / timespan.Seconds()
}
