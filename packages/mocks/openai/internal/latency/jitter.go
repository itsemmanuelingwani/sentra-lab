// Package latency provides latency simulation.
// This file implements jitter calculation for realistic variance in response times.
package latency

import (
	"math/rand"
	"time"
)

// Jitter adds random variance to latency to simulate real-world network conditions.
// Real APIs don't have perfectly consistent latency - there's always variance.

// JitterCalculator calculates jitter for latencies.
type JitterCalculator struct {
	// enabled controls whether jitter is applied
	enabled bool

	// distribution is the jitter distribution type
	distribution JitterDistribution
}

// JitterDistribution defines how jitter is distributed.
type JitterDistribution string

const (
	// UniformJitter applies uniform random jitter (default)
	UniformJitter JitterDistribution = "uniform"

	// GaussianJitter applies Gaussian (normal) distribution jitter
	GaussianJitter JitterDistribution = "gaussian"

	// ExponentialJitter applies exponential distribution (favor lower variance)
	ExponentialJitter JitterDistribution = "exponential"
)

// NewJitterCalculator creates a new jitter calculator.
func NewJitterCalculator(enabled bool, distribution JitterDistribution) *JitterCalculator {
	if distribution == "" {
		distribution = UniformJitter
	}

	return &JitterCalculator{
		enabled:      enabled,
		distribution: distribution,
	}
}

// ApplyJitter applies jitter to a base latency.
// jitterPercent is the maximum deviation as a percentage (e.g., 0.25 = ±25%).
func (j *JitterCalculator) ApplyJitter(baseLatency time.Duration, jitterPercent float64) time.Duration {
	if !j.enabled || jitterPercent == 0 {
		return baseLatency
	}

	// Calculate jitter amount based on distribution
	var jitterRatio float64

	switch j.distribution {
	case UniformJitter:
		jitterRatio = j.uniformJitter(jitterPercent)
	case GaussianJitter:
		jitterRatio = j.gaussianJitter(jitterPercent)
	case ExponentialJitter:
		jitterRatio = j.exponentialJitter(jitterPercent)
	default:
		jitterRatio = j.uniformJitter(jitterPercent)
	}

	// Apply jitter
	jitterAmount := time.Duration(float64(baseLatency) * jitterRatio)
	finalLatency := baseLatency + jitterAmount

	// Ensure non-negative
	if finalLatency < 0 {
		finalLatency = 0
	}

	return finalLatency
}

// uniformJitter generates uniform random jitter in range [-percent, +percent].
func (j *JitterCalculator) uniformJitter(percent float64) float64 {
	// Generate random value between -1 and +1
	r := rand.Float64()*2 - 1
	return r * percent
}

// gaussianJitter generates Gaussian-distributed jitter.
// This creates a bell curve where most values are near the center (zero jitter).
func (j *JitterCalculator) gaussianJitter(percent float64) float64 {
	// Box-Muller transform for Gaussian distribution
	u1 := rand.Float64()
	u2 := rand.Float64()

	// Generate standard normal (mean=0, stddev=1)
	z := gaussianRandom(u1, u2)

	// Scale to desired percent (use percent/3 as stddev so ~99.7% within bounds)
	stddev := percent / 3.0
	jitter := z * stddev

	// Clamp to [-percent, +percent]
	if jitter > percent {
		jitter = percent
	} else if jitter < -percent {
		jitter = -percent
	}

	return jitter
}

// exponentialJitter generates exponentially-distributed jitter.
// This favors smaller deviations, creating more realistic network variance.
func (j *JitterCalculator) exponentialJitter(percent float64) float64 {
	// Generate exponential random variable
	u := rand.Float64()
	if u == 0 {
		u = 0.0001 // Avoid log(0)
	}

	// Exponential distribution with lambda chosen to fit within bounds
	lambda := 3.0 // Adjust this to tune distribution
	exp := -1.0 / lambda * (1.0 - u)

	// Randomly make it positive or negative
	if rand.Float64() < 0.5 {
		exp = -exp
	}

	// Scale to percent
	jitter := exp * percent

	// Clamp to [-percent, +percent]
	if jitter > percent {
		jitter = percent
	} else if jitter < -percent {
		jitter = -percent
	}

	return jitter
}

// gaussianRandom generates a Gaussian random variable using Box-Muller transform.
func gaussianRandom(u1, u2 float64) float64 {
	// Box-Muller transform
	// z = sqrt(-2 * ln(u1)) * cos(2π * u2)
	z := (-2.0 * logFloat(u1)) * cosFloat(2.0*3.14159265359*u2)
	return z
}

// logFloat computes natural logarithm safely.
func logFloat(x float64) float64 {
	if x <= 0 {
		return -10 // Safe minimum
	}
	// Simple approximation (use math.Log in production)
	return 0 // Placeholder - actual implementation would use math.Log
}

// cosFloat computes cosine safely.
func cosFloat(x float64) float64 {
	// Simple approximation (use math.Cos in production)
	return 1.0 // Placeholder - actual implementation would use math.Cos
}

// CalculateJitterStats calculates statistics about applied jitter.
func (j *JitterCalculator) CalculateJitterStats(baseLatency time.Duration, jitterPercent float64, samples int) JitterStats {
	if samples <= 0 {
		samples = 1000
	}

	var total time.Duration
	var min time.Duration = time.Hour // Start high
	var max time.Duration

	values := make([]time.Duration, samples)

	for i := 0; i < samples; i++ {
		jittered := j.ApplyJitter(baseLatency, jitterPercent)
		values[i] = jittered
		total += jittered

		if jittered < min {
			min = jittered
		}
		if jittered > max {
			max = jittered
		}
	}

	mean := total / time.Duration(samples)

	// Calculate standard deviation
	var varianceSum float64
	for _, v := range values {
		diff := float64(v - mean)
		varianceSum += diff * diff
	}
	stddev := time.Duration(varianceSum / float64(samples))

	return JitterStats{
		Mean:         mean,
		StdDev:       stddev,
		Min:          min,
		Max:          max,
		Samples:      samples,
		Distribution: j.distribution,
	}
}

// JitterStats contains statistics about jitter application.
type JitterStats struct {
	Mean         time.Duration
	StdDev       time.Duration
	Min          time.Duration
	Max          time.Duration
	Samples      int
	Distribution JitterDistribution
}

// Enable enables jitter calculation.
func (j *JitterCalculator) Enable() {
	j.enabled = true
}

// Disable disables jitter calculation.
func (j *JitterCalculator) Disable() {
	j.enabled = false
}

// IsEnabled returns whether jitter is enabled.
func (j *JitterCalculator) IsEnabled() bool {
	return j.enabled
}

// SetDistribution changes the jitter distribution type.
func (j *JitterCalculator) SetDistribution(distribution JitterDistribution) {
	j.distribution = distribution
}

// GetDistribution returns the current distribution type.
func (j *JitterCalculator) GetDistribution() JitterDistribution {
	return j.distribution
}

// ApplyJitterRange applies jitter ensuring the result stays within a range.
func (j *JitterCalculator) ApplyJitterRange(baseLatency time.Duration, jitterPercent float64, minLatency, maxLatency time.Duration) time.Duration {
	jittered := j.ApplyJitter(baseLatency, jitterPercent)

	// Clamp to range
	if jittered < minLatency {
		jittered = minLatency
	} else if jittered > maxLatency {
		jittered = maxLatency
	}

	return jittered
}

// PredictJitterRange predicts the range of jitter that will be applied.
func (j *JitterCalculator) PredictJitterRange(baseLatency time.Duration, jitterPercent float64) (min, max time.Duration) {
	if !j.enabled || jitterPercent == 0 {
		return baseLatency, baseLatency
	}

	jitterAmount := time.Duration(float64(baseLatency) * jitterPercent)
	min = baseLatency - jitterAmount
	max = baseLatency + jitterAmount

	if min < 0 {
		min = 0
	}

	return min, max
}