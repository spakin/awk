package awk

import (
	"fmt"
	"strconv"
)

const convFmt = "%.6g"

// A Value represents an immutable datum that can be converted to various
// types, always without error.
type Value struct {
	i64 int64   // Value converted to an int64
	f64 float64 // Value converted to a float64
	s   string  // Value converted to a string

	i64_ok bool // true: i64 is valid; false: invalid
	f64_ok bool // true: f64 is valid; false: invalid
	s_ok   bool // true: s is valid; false: invalid
}

// NewInt64 creates a Value from an Int64.
func NewInt64(i int64) *Value {
	return &Value{i64: i, i64_ok: true}
}

// Int64 converts a Value to an int64.  This method always succeeds.
func (v *Value) Int64() int64 {
	switch {
	case v.f64_ok:
		v.i64 = int64(v.f64)
		v.i64_ok = true
	case v.s_ok:
		v.i64, _ = strconv.ParseInt(v.s, 10, 64)
		v.i64_ok = true
	}
	return v.i64
}

// Float64 converts a Value to a float64.  This method always succeeds.
func (v *Value) Float64() float64 {
	switch {
	case v.i64_ok:
		v.f64 = float64(v.i64)
		v.f64_ok = true
	case v.s_ok:
		v.f64, _ = strconv.ParseFloat(v.s, 64)
		v.f64_ok = true
	}
	return v.f64
}

// String converts a Value to a string.  This method always succeeds.
func (v *Value) String() string {
	switch {
	case v.i64_ok:
		v.s = strconv.FormatInt(v.i64, 10)
		v.s_ok = true
	case v.f64_ok:
		v.s = fmt.Sprintf(convFmt, v.f64)
		v.s_ok = true
	}
	return v.s
}
