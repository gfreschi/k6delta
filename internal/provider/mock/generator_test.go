package mock

import (
	"math"
	"testing"
)

func TestConstant(t *testing.T) {
	g := Constant(42.0)
	for _, tc := range []float64{0.0, 0.5, 1.0} {
		if got := g(tc); got != 42.0 {
			t.Errorf("Constant(42)(%.1f) = %v, want 42", tc, got)
		}
	}
}

func TestSine(t *testing.T) {
	g := Sine(50.0, 30.0)
	// At t=0: sin(0)=0, so base=50
	if got := g(0.0); math.Abs(got-50.0) > 0.01 {
		t.Errorf("Sine(50,30)(0.0) = %v, want ~50", got)
	}
	// At t=0.25: sin(pi/2)=1, so base+amplitude=80
	if got := g(0.25); math.Abs(got-80.0) > 0.01 {
		t.Errorf("Sine(50,30)(0.25) = %v, want ~80", got)
	}
}

func TestRamp(t *testing.T) {
	g := Ramp(10.0, 90.0)
	if got := g(0.0); got != 10.0 {
		t.Errorf("Ramp(10,90)(0.0) = %v, want 10", got)
	}
	if got := g(0.5); got != 50.0 {
		t.Errorf("Ramp(10,90)(0.5) = %v, want 50", got)
	}
	if got := g(1.0); got != 90.0 {
		t.Errorf("Ramp(10,90)(1.0) = %v, want 90", got)
	}
}

func TestStep(t *testing.T) {
	g := Step(2.0, 5.0, 0.5)
	if got := g(0.3); got != 2.0 {
		t.Errorf("Step(2,5,0.5)(0.3) = %v, want 2", got)
	}
	if got := g(0.7); got != 5.0 {
		t.Errorf("Step(2,5,0.5)(0.7) = %v, want 5", got)
	}
}

func TestNoise(t *testing.T) {
	base := Constant(50.0)
	g := Noise(base, 5.0)
	// All values should be within [45, 55]
	for i := 0; i < 100; i++ {
		v := g(float64(i) / 100.0)
		if v < 45.0 || v > 55.0 {
			t.Errorf("Noise(Constant(50), 5)(%.2f) = %v, want [45,55]", float64(i)/100.0, v)
		}
	}
}

func TestSample(t *testing.T) {
	g := Constant(42.0)
	values := Sample(g, 10)
	if len(values) != 10 {
		t.Fatalf("Sample(_, 10) returned %d values, want 10", len(values))
	}
	for i, v := range values {
		if v != 42.0 {
			t.Errorf("values[%d] = %v, want 42", i, v)
		}
	}
}
