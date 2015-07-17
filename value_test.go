package awk

import (
	"math"
	"testing"
)

// TestInt64ToInt64 converts various int64s to Values then back to int64s.
func TestInt64ToInt64(t *testing.T) {
	scr := NewScript()
	for _, n := range []int64{0, -123, 123, -456, 456, math.MaxInt64, math.MinInt64, 123} {
		v := scr.NewInt64(n)
		i := v.Int64()
		if i != n {
			t.Fatalf("Expected %d but received %d", n, i)
		}
	}
}

// TestInt64ToInt64 converts various int64s to Values then to float64s.
func TestInt64ToFloat64(t *testing.T) {
	scr := NewScript()
	for _, n := range []int64{0, -123, 123, -456, 456, math.MaxInt64, math.MinInt64, 123} {
		v := scr.NewInt64(n)
		f := v.Float64()
		if f != float64(n) {
			t.Fatalf("Expected %.4g but received %.4g", float64(n), f)
		}
	}
}

// TestInt64ToString converts various int64s to Values then to strings.
func TestInt64ToString(t *testing.T) {
	scr := NewScript()
	in := []int64{0, -123, 123, -456, 456, math.MaxInt64, math.MinInt64, 123}
	out := []string{"0", "-123", "123", "-456", "456", "9223372036854775807", "-9223372036854775808", "123"}
	for i, n := range in {
		v := scr.NewInt64(n)
		o := v.String()
		if o != out[i] {
			t.Fatalf("Expected \"%d\" but received %q", n, out[i])
		}
	}
}
