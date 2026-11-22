package randomness

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// describes one meal period in time-of-day durations
type ArrivalWindow struct {
	Count                    int
	Open, Close, Peak        time.Duration
	EarlyLead                time.Duration
	SigmaLeft, SigmaRight    time.Duration
	EarlyRatio, UniformRatio float64
	StopBeforeClose          time.Duration
}

type arrivalSession struct {
	config                           ArrivalWindow
	earlyRemaining, regularRemaining int
}

type ArrivalProcess struct {
	rng      *rand.Rand
	sessions [2]arrivalSession
}

func NewArrivalProcess(windows [2]ArrivalWindow, start time.Duration, seed int64) (*ArrivalProcess, error) {
	p := &ArrivalProcess{rng: rand.New(rand.NewSource(seed))}
	for i, w := range windows {
		cutoff := w.Close - w.StopBeforeClose

		if w.Count < 0 ||
			w.Open < 0 ||
			w.StopBeforeClose < 0 ||
			cutoff <= w.Open ||
			w.Peak <= w.Open ||
			w.Peak >= cutoff ||
			w.EarlyLead <= 0 ||
			w.SigmaLeft <= 0 ||
			w.SigmaRight <= 0 ||
			math.IsNaN(w.EarlyRatio) ||
			math.IsNaN(w.UniformRatio) ||
			w.EarlyRatio < 0 || w.EarlyRatio > 1 ||
			w.UniformRatio < 0 || w.UniformRatio > 1 {
			return nil, fmt.Errorf("invalid arrival window %d", i)
		}

		if i == 1 && windows[0].Close > w.Open-w.EarlyLead {
			return nil, fmt.Errorf("lunch and dinner arrival windows overlap")
		}
		early := 0
		if w.EarlyRatio > 0 && w.Count > 0 {
			early = max(1, int(math.Round(float64(w.Count)*w.EarlyRatio)))
			early = min(early, w.Count)
		}
		regular := w.Count - early
		// If a simulation starts partway through service, skip past arrivals.
		p.sessions[i] = arrivalSession{
			config:           w,
			earlyRemaining:   drawBinomial(p.rng, early, 1-earlyCDF(w, start)),
			regularRemaining: drawBinomial(p.rng, regular, 1-regularCDF(w, start)),
		}
	}
	return p, nil
}

// how many people will arrive next step? -> lunch[0] dinner[1]
func (p *ArrivalProcess) Next(from, to time.Duration) [2]int {
	var counts [2]int
	if to <= from {
		return counts
	}
	for i := range p.sessions {
		s := &p.sessions[i]
		a0, a1 := earlyCDF(s.config, from), earlyCDF(s.config, to)
		b0, b1 := regularCDF(s.config, from), regularCDF(s.config, to)
		early := drawBinomial(p.rng, s.earlyRemaining, conditionalProbability(a0, a1))
		regular := drawBinomial(p.rng, s.regularRemaining, conditionalProbability(b0, b1))
		s.earlyRemaining -= early
		s.regularRemaining -= regular
		counts[i] = early + regular
	}
	return counts
}

func conditionalProbability(before, after float64) float64 {
	if before >= 1 {
		return 0
	}
	return max(0, min(1, (after-before)/(1-before)))
}

func earlyCDF(w ArrivalWindow, t time.Duration) float64 {
	begin := w.Open - w.EarlyLead
	return max(0, min(1, float64(t-begin)/float64(w.EarlyLead)))
}

// F(t) = P(T <= t),  T is a regular student's arrival time
// uniform background + truncated asymmetric Gaussian.
func regularCDF(w ArrivalWindow, t time.Duration) float64 {
	cutoff := w.Close - w.StopBeforeClose

	if t <= w.Open {
		return 0
	}
	if t >= cutoff {
		return 1
	}

	uniform := float64(t-w.Open) / float64(cutoff-w.Open)

	gaussian := gaussianArea(w, w.Open, t) /
		gaussianArea(w, w.Open, cutoff)

	return max(0, min(1,
		w.UniformRatio*uniform+
			(1-w.UniformRatio)*gaussian,
	))
}
func gaussianArea(w ArrivalWindow, from, to time.Duration) float64 {
	if from >= to {
		return 0
	}
	piece := func(a, b, spread time.Duration) float64 {
		if a >= b {
			return 0
		}
		sigma := float64(spread)
		center := float64(w.Peak)
		return sigma * math.Sqrt(math.Pi/2) * (math.Erf((float64(b)-center)/(sigma*math.Sqrt2)) -
			math.Erf((float64(a)-center)/(sigma*math.Sqrt2)))
	}
	return piece(from, min(to, w.Peak), w.SigmaLeft) +
		piece(max(from, w.Peak), to, w.SigmaRight)
}

// Draw a binomial sample using geometric jumps; efficient for tiny tick probabilities.
func drawBinomial(rng *rand.Rand, n int, probability float64) int {
	if n <= 0 || probability <= 0 {
		return 0
	}
	if probability >= 1 {
		return n
	}
	if probability > 0.5 {
		return n - drawBinomial(rng, n, 1-probability)
	}
	logFailure := math.Log1p(-probability)
	seen, successes := 0, 0
	for seen < n {
		failures := math.Floor(math.Log1p(-rng.Float64()) / logFailure)
		if failures >= float64(n-seen) {
			break
		}
		seen += int(failures) + 1
		successes++
	}
	return successes
}
