// Package mock implements a synthetic InfraProvider for testing and demos.
package mock

import (
	"math"
	"math/rand/v2"
)

// Generator produces a value for a normalized time t in [0.0, 1.0].
type Generator func(t float64) float64

// Constant returns the same value regardless of time.
func Constant(v float64) Generator {
	return func(_ float64) float64 { return v }
}

// Sine produces a sinusoidal wave centered on base with the given amplitude.
// One full cycle spans t=[0,1].
func Sine(base, amplitude float64) Generator {
	return func(t float64) float64 {
		return base + amplitude*math.Sin(2*math.Pi*t)
	}
}

// Ramp linearly interpolates from start to end over t=[0,1].
func Ramp(from, to float64) Generator {
	return func(t float64) float64 {
		return from + (to-from)*t
	}
}

// Step returns before when t < at, and after when t >= at.
func Step(before, after, at float64) Generator {
	return func(t float64) float64 {
		if t < at {
			return before
		}
		return after
	}
}

// Noise adds random jitter in [-jitter, +jitter] to a base generator.
func Noise(base Generator, jitter float64) Generator {
	return func(t float64) float64 {
		return base(t) + (rand.Float64()*2-1)*jitter
	}
}

// Sample evaluates a generator at n evenly-spaced points over [0, 1].
func Sample(g Generator, n int) []float64 {
	values := make([]float64, n)
	for i := range n {
		var t float64
		if n == 1 {
			t = 0.5
		} else {
			t = float64(i) / float64(n-1)
		}
		values[i] = g(t)
	}
	return values
}
