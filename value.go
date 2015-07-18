// This file defines an AWK-like data type, Value, that can easily be converted
// to different Go data types.

package awk

import (
	"fmt"
	"strconv"
)

const convFmt = "%.6g"

// A Value represents an immutable datum that can be converted to an int,
// float64, or string in best-effort fashion (i.e., never returning an error).
type Value struct {
	ival int     // Value converted to an int
	fval float64 // Value converted to a float64
	sval string  // Value converted to a string

	ival_ok bool // true: ival is valid; false: invalid
	fval_ok bool // true: fval is valid; false: invalid
	sval_ok bool // true: sval is valid; false: invalid

	script *Script // Pointer to the script that produced this value
}

// NewValue creates a Value from an arbitrary Go data type.  Data types that do
// not map straightforwardly to one of {int, float64, string} are represented
// by a zero value.
func (s *Script) NewValue(v interface{}) *Value {
	val := &Value{script: s}
	switch v := v.(type) {
	case uint:
		val.ival = int(v)
		val.ival_ok = true
	case uint8:
		val.ival = int(v)
		val.ival_ok = true
	case uint16:
		val.ival = int(v)
		val.ival_ok = true
	case uint32:
		val.ival = int(v)
		val.ival_ok = true
	case uint64:
		val.ival = int(v)
		val.ival_ok = true
	case uintptr:
		val.ival = int(v)
		val.ival_ok = true

	case int:
		val.ival = int(v)
		val.ival_ok = true
	case int8:
		val.ival = int(v)
		val.ival_ok = true
	case int16:
		val.ival = int(v)
		val.ival_ok = true
	case int32:
		val.ival = int(v)
		val.ival_ok = true
	case int64:
		val.ival = int(v)
		val.ival_ok = true

	case float32:
		val.fval = float64(v)
		val.fval_ok = true
	case float64:
		val.fval = float64(v)
		val.fval_ok = true

	case complex64:
		val.fval = float64(real(v))
		val.fval_ok = true
	case complex128:
		val.fval = float64(real(v))
		val.fval_ok = true

	case string:
		val.sval = v
		val.sval_ok = true

	default:
		val.sval_ok = true
	}
	return val
}

// Int converts a Value to an int.
func (v *Value) Int() int {
	switch {
	case v.fval_ok:
		v.ival = int(v.fval)
		v.ival_ok = true
	case v.sval_ok:
		i64, _ := strconv.ParseInt(v.sval, 10, 0)
		v.ival = int(i64)
		v.ival_ok = true
	}
	return v.ival
}

// Float64 converts a Value to a float64.
func (v *Value) Float64() float64 {
	switch {
	case v.ival_ok:
		v.fval = float64(v.ival)
		v.fval_ok = true
	case v.sval_ok:
		v.fval, _ = strconv.ParseFloat(v.sval, 64)
		v.fval_ok = true
	}
	return v.fval
}

// String converts a Value to a string.
func (v *Value) String() string {
	switch {
	case v.ival_ok:
		v.sval = strconv.FormatInt(int64(v.ival), 10)
		v.sval_ok = true
	case v.fval_ok:
		v.sval = fmt.Sprintf(v.script.ConvFmt, v.fval)
		v.sval_ok = true
	}
	return v.sval
}
