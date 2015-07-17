package awk

import (
	"math"
	"testing"
)

// TestInt64ToInt64 converts various int64s to Values then back to int64s.
func TestInt64ToInt64(t *testing.T) {
	for _, n := range []int64{0, -123, 123, -456, 456, math.MaxInt64, math.MinInt64, 123} {
		v := NewInt64(n)
		i := v.Int64()
		if i != n {
			t.Fatalf("Expected %d but received %d", n, i)
		}
	}
}

// TestInt64ToInt64 converts various int64s to Values then to float64s.
func TestInt64ToFloat64(t *testing.T) {
	for _, n := range []int64{0, -123, 123, -456, 456, math.MaxInt64, math.MinInt64, 123} {
		v := NewInt64(n)
		f := v.Float64()
		if f != float64(n) {
			t.Fatalf("Expected %.4g but received %.4g", float64(n), f)
		}
	}
}

// TestInt64ToString converts various int64s to Values then to strings.
func TestInt64ToString(t *testing.T) {
}
