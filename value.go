// This file defines an AWK-like data type, Value, that can easily be converted
// to different Go data types.

package awk

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const convFmt = "%.6g"

// A Value represents an immutable datum that can be converted to an int,
// float64, or string in best-effort fashion (i.e., never returning an error).
type Value struct {
	ival int     // Value converted to an int
	fval float64 // Value converted to a float64
	sval string  // Value converted to a string

	ivalOk bool // true: ival is valid; false: invalid
	fvalOk bool // true: fval is valid; false: invalid
	svalOk bool // true: sval is valid; false: invalid

	script *Script // Pointer to the script that produced this value
}

// NewValue creates a Value from an arbitrary Go data type.  Data types that do
// not map straightforwardly to one of {int, float64, string} are represented
// by a zero value.
func (s *Script) NewValue(v interface{}) *Value {
	val := &Value{}
	switch v := v.(type) {
	case uint:
		val.ival = int(v)
		val.ivalOk = true
	case uint8:
		val.ival = int(v)
		val.ivalOk = true
	case uint16:
		val.ival = int(v)
		val.ivalOk = true
	case uint32:
		val.ival = int(v)
		val.ivalOk = true
	case uint64:
		val.ival = int(v)
		val.ivalOk = true
	case uintptr:
		val.ival = int(v)
		val.ivalOk = true

	case int:
		val.ival = int(v)
		val.ivalOk = true
	case int8:
		val.ival = int(v)
		val.ivalOk = true
	case int16:
		val.ival = int(v)
		val.ivalOk = true
	case int32:
		val.ival = int(v)
		val.ivalOk = true
	case int64:
		val.ival = int(v)
		val.ivalOk = true

	case bool:
		if v {
			val.ival = 1
		}
		val.ivalOk = true

	case float32:
		val.fval = float64(v)
		val.fvalOk = true
	case float64:
		val.fval = float64(v)
		val.fvalOk = true

	case complex64:
		val.fval = float64(real(v))
		val.fvalOk = true
	case complex128:
		val.fval = float64(real(v))
		val.fvalOk = true

	case string:
		val.sval = v
		val.svalOk = true

	case *Value:
		*val = *v

	default:
		val.svalOk = true
	}
	val.script = s
	return val
}

// matchInt matches a base-ten integer.
var matchInt = regexp.MustCompile(`^\s*([-+]?\d+)`)

// Int converts a Value to an int.
func (v *Value) Int() int {
	switch {
	case v.ivalOk:
	case v.fvalOk:
		v.ival = int(v.fval)
		v.ivalOk = true
	case v.svalOk:
		// Perform a best-effort conversion from string to int.
		strs := matchInt.FindStringSubmatch(v.sval)
		var i64 int64
		if len(strs) >= 2 {
			i64, _ = strconv.ParseInt(strs[1], 10, 0)
		}
		v.ival = int(i64)
		v.ivalOk = true
	}
	return v.ival
}

// matchFloat matches a base-ten floating-point number.
var matchFloat = regexp.MustCompile(`^\s*([-+]?(?:\d+(?:\.\d*)?|\.\d+)(?:[Ee][-+]?\d+)?)`)

// Float64 converts a Value to a float64.
func (v *Value) Float64() float64 {
	switch {
	case v.fvalOk:
	case v.ivalOk:
		v.fval = float64(v.ival)
		v.fvalOk = true
	case v.svalOk:
		// Perform a best-effort conversion from string to float64.
		v.fval = 0.0
		strs := matchFloat.FindStringSubmatch(v.sval)
		if len(strs) >= 2 {
			v.fval, _ = strconv.ParseFloat(strs[1], 64)
		}
		v.fvalOk = true
	}
	return v.fval
}

// String converts a Value to a string.
func (v *Value) String() string {
	switch {
	case v.svalOk:
	case v.ivalOk:
		v.sval = strconv.FormatInt(int64(v.ival), 10)
		v.svalOk = true
	case v.fvalOk:
		v.sval = fmt.Sprintf(v.script.ConvFmt, v.fval)
		v.svalOk = true
	}
	return v.sval
}

// Match says whether a given regular expression, provided as a string, matches
// the Value.  If the associated script set IgnoreCase(true), the match is
// tested in a case-insensitive manner.
func (v *Value) Match(expr string) bool {
	// Compile the regular expression.
	re, err := v.script.compileRegexp(expr)
	if err != nil {
		return false // Fail silently
	}

	// Return true if the expression matches the value, interpreted as a
	// string.
	loc := re.FindStringIndex(v.String())
	if loc == nil {
		v.script.RStart = 0
		v.script.RLength = -1
		return false
	}
	v.script.RStart = loc[0] + 1
	v.script.RLength = loc[1] - loc[0]
	return true
}

// StrEqual says whether a Value, treated as a string, has the same contents as
// a given Value, which can be provided either as a Value or as any type that
// can be converted to a Value.  If the associated script called
// IgnoreCase(true), the comparison is performed in a case-insensitive manner.
func (v *Value) StrEqual(v2 interface{}) bool {
	switch v2 := v2.(type) {
	case *Value:
		if v.script.ignCase {
			return strings.EqualFold(v.String(), v2.String())
		} else {
			return v.String() == v2.String()
		}
	case string:
		if v.script.ignCase {
			return strings.EqualFold(v.String(), v2)
		} else {
			return v.String() == v2
		}
	default:
		v2Val := v.script.NewValue(v2)
		if v.script.ignCase {
			return strings.EqualFold(v.String(), v2Val.String())
		} else {
			return v.String() == v2Val.String()
		}
	}
}
