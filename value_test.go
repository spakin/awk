// This file tests conversions from each data type to every other data type.

package awk

import (
	"math"
	"testing"
)

// TestIntToInt converts various ints to Values then back to ints.
func TestIntToInt(t *testing.T) {
	scr := NewScript()
	for _, n := range []int{0, -123, 123, -456, 456, math.MaxInt32, math.MinInt32, 123} {
		v := scr.NewValue(n)
		i := v.Int()
		if i != n {
			t.Fatalf("Expected %d but received %d", n, i)
		}
	}
}

// TestIntToInt converts various ints to Values then to float64s.
func TestIntToFloat64(t *testing.T) {
	scr := NewScript()
	for _, n := range []int{0, -123, 123, -456, 456, math.MaxInt32, math.MinInt32, 123} {
		v := scr.NewValue(n)
		f := v.Float64()
		if f != float64(n) {
			t.Fatalf("Expected %.4g but received %.4g", float64(n), f)
		}
	}
}

// TestIntToString converts various ints to Values then to strings.
func TestIntToString(t *testing.T) {
	scr := NewScript()
	in := []int{0, -123, 123, -456, 456, math.MaxInt32, math.MinInt32, 123}
	out := []string{"0", "-123", "123", "-456", "456", "2147483647", "-2147483648", "123"}
	for idx, n := range in {
		v := scr.NewValue(n)
		s := v.String()
		if s != out[idx] {
			t.Fatalf("Expected %q but received %q", out[idx], s)
		}
	}
}

// TestFloat64ToInt converts various float64s to Values then to ints.
func TestFloat64ToInt(t *testing.T) {
	scr := NewScript()
	in := []float64{0.0, -123.0, 123.0, -456.7, 456.7, 123.0, -456.4, 456.4}
	out := []int{0, -123, 123, -456, 456, 123, -456, 456}
	for idx, n := range in {
		v := scr.NewValue(n)
		i := v.Int()
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
		v := scr.NewValue(n)
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
		v := scr.NewValue(n)
		s := v.String()
		if s != out[idx] {
			t.Fatalf("Expected %q but received %q", out[idx], s)
		}
	}
}

// TestStringToInt converts various strings to Values then to ints.
func TestStringToInt(t *testing.T) {
	scr := NewScript()
	in := []string{"0", "-123", "123", "-456", "456", "9223372036854775807", "-9223372036854775808", "123", "Text999", "321_go"}
	out := []int{0, -123, 123, -456, 456, 9223372036854775807, -9223372036854775808, 123, 0, 321}
	for idx, n := range in {
		v := scr.NewValue(n)
		i := v.Int()
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
		v := scr.NewValue(n)
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
		v := scr.NewValue(n)
		s := v.String()
		if s != n {
			t.Fatalf("Expected %q but received %q", n, s)
		}
	}
}

// TestMatch tests if regular-expression matching works.
func TestMatch(t *testing.T) {
	// We run the test twice to confirm that regexp caching works.
	scr := NewScript()
	v := scr.NewValue("Mississippi")
	in := []string{"p*", "[is]+", "Miss", "hippie", "ippi"}
	out := []bool{true, true, true, false, true}
	for range [2]struct{}{} {
		for idx, n := range in {
			m := v.Match(n)
			if m != out[idx] {
				t.Fatalf("Expected %v but received %v\n", out[idx], m)
			}
		}
	}

	// Test if RStart and RLength are maintained properly.
	if !v.Match("[is]+") {
		t.Fatalf("Failed to match %v against %q", v, "[is]+")
	}
	if scr.RStart != 2 || scr.RLength != 7 {
		t.Fatalf("Expected {2, 7} but received {%d, %d}", scr.RStart, scr.RLength)
	}
	if v.Match("[xy]+") {
		t.Fatalf("Incorrectly matched %v against %q", v, "[xy]+")
	}
	if scr.RStart != 0 || scr.RLength != -1 {
		t.Fatalf("Expected {0, -1} but received {%d, %d}", scr.RStart, scr.RLength)
	}
}

// TestStrEqual tests if string comparisons work.
func TestStrEqual(t *testing.T) {
	// Test case-sensitive comparisons.
	scr := NewScript()
	v := scr.NewValue("good")
	for _, bad := range []string{"bad", "goody", "Good", "good "} {
		if v.StrEqual(scr.NewValue(bad)) {
			t.Fatalf("Incorrectly matched %q = %q", "good", bad)
		}
	}
	if !v.StrEqual(scr.NewValue("good")) {
		t.Fatalf("Failed to match %q", "good")
	}

	// Test case-insensitive comparisons.
	scr.IgnoreCase(true)
	for _, bad := range []string{"bad", "goody", "good "} {
		if v.StrEqual(scr.NewValue(bad)) {
			t.Fatalf("Incorrectly matched %q = %q", "good", bad)
		}
	}
	if !v.StrEqual(scr.NewValue("good")) {
		t.Fatalf("Failed to match %q", "good")
	}
	if !v.StrEqual(scr.NewValue("GooD")) {
		t.Fatalf("Failed to match %q = %q", "good", "GooD")
	}
}
