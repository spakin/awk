// This file tests conversions from each data type to every other data type.

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
	for idx, n := range in {
		v := scr.NewInt64(n)
		s := v.String()
		if s != out[idx] {
			t.Fatalf("Expected %q but received %q", out[idx], s)
		}
	}
}

// TestFloat64ToInt64 converts various float64s to Values then to int64s.
func TestFloat64ToInt64(t *testing.T) {
	scr := NewScript()
	in := []float64{0.0, -123.0, 123.0, -456.7, 456.7, math.MaxFloat64, -math.MaxFloat64, 123.0, -456.4, 456.4}
	out := []int64{0, -123, 123, -456, 456, math.MinInt64, math.MinInt64, 123, -456, 456}
	for idx, n := range in {
		v := scr.NewFloat64(n)
		i := v.Int64()
		if i != out[idx] {
			t.Fatalf("Expected %d but received %d", out[idx], i)
		}
	}
}

// TestFloat64ToFloat64 converts various float64s to Values then back to
// float64s.
func TestFloat64ToFloat64(t *testing.T) {
	scr := NewScript()
	for _, n := range []float64{0.0, -123.0, 123.0, -456.7, 456.7, math.MaxFloat64, -math.MaxFloat64, 123.0, -456.4, 456.4} {
		v := scr.NewFloat64(n)
		f := v.Float64()
		if f != n {
			t.Fatalf("Expected %.4g but received %.4g", n, f)
		}
	}
}

// TestFloat64ToString converts various float64s to Values then to strings.
func TestFloat64ToString(t *testing.T) {
	scr := NewScript()
	in := []float64{0.0, -123.0, 123.0, -456.7, 456.7, math.MaxFloat64, -math.MaxFloat64, 123.0, -456.4, 456.4}
	out := []string{"0", "-123", "123", "-456.7", "456.7", "1.79769e+308", "-1.79769e+308", "123", "-456.4", "456.4"}
	for idx, n := range in {
		v := scr.NewFloat64(n)
		s := v.String()
		if s != out[idx] {
			t.Fatalf("Expected %q but received %q", out[idx], s)
		}
	}
}

// TestStringToInt64 converts various strings to Values then to int64s.
func TestStringToInt64(t *testing.T) {
	scr := NewScript()
	in := []string{"0", "-123", "123", "-456", "456", "9223372036854775807", "-9223372036854775808", "123", "Text999"}
	out := []int64{0, -123, 123, -456, 456, 9223372036854775807, -9223372036854775808, 123, 0}
	for idx, n := range in {
		v := scr.NewString(n)
		i := v.Int64()
		if i != out[idx] {
			t.Fatalf("Expected %d but received %d", out[idx], i)
		}
	}
}

// TestStringToFloat64 converts various strings to Values then to float64s.
func TestStringToFloat64(t *testing.T) {
	scr := NewScript()
	in := []string{"0", "-123", "123", "-456.7", "456.7", "17.9769e+307", "-17.9769e+307", "123", "-456.4", "456.4", "Text99.99", "99.99e+1000"}
	out := []float64{0, -123, 123, -456.7, 456.7, 1.79769e+308, -1.79769e+308, 123, -456.4, 456.4, 0, math.Inf(1)}
	for idx, n := range in {
		v := scr.NewString(n)
		f := v.Float64()
		if f != out[idx] {
			t.Fatalf("Expected %.4g but received %.4g", out[idx], f)
		}
	}
}

// TestStringToString converts various strings to Values then back to strings.
func TestStringToString(t *testing.T) {
	scr := NewScript()
	for _, n := range []string{"0", "-123", "123", "-456.7", "456.7", "17.9769e+307", "-17.9769e+307", "123", "-456.4", "456.4", "Text99.99", "99.99e+1000"} {
		v := scr.NewString(n)
		s := v.String()
		if s != n {
			t.Fatalf("Expected %q but received %q", n, s)
		}
	}
}
