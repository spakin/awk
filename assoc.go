// This file defines an AWK-like associative array, ValueArray.

package awk

import (
	"strings"
)

// A ValueArray maps Values to Values.
type ValueArray struct {
	script *Script           // Pointer to the script that produced this value
	data   map[string]*Value // The associative array proper
}

// NewValueArray creates and returns an associative array of Values.
func (s *Script) NewValueArray() *ValueArray {
	return &ValueArray{
		script: s,
		data:   make(map[string]*Value),
	}
}

// Set (index, value) assigns a Value to an index of a ValueArray.  Multiple
// indexes can be specified to simulate multidimensional arrays.  (In fact, the
// indexes are concatenated into a single string with intervening Script.SubSep
// characters.)  The final argument is always the value to assign.  Arguments
// can be provided either as Values or as any types that can be converted to
// Values.
func (va *ValueArray) Set(args ...interface{}) {
	// Ensure we were given at least one index and a value.
	if len(args) < 2 {
		panic("ValueArray.Set requires at least one index and one value")
	}

	// Convert each argument to a Value.
	argVals := make([]*Value, len(args))
	for i, arg := range args {
		v, ok := arg.(*Value)
		if !ok {
			v = va.script.NewValue(arg)
		}
		argVals[i] = v
	}

	// Handle the most common case: one index and one value.
	if len(args) == 2 {
		va.data[argVals[0].String()] = argVals[1]
		return
	}

	// Merge the indexes into a single string.
	idxStrs := make([]string, len(argVals)-1)
	for i, v := range argVals[:len(argVals)-1] {
		idxStrs[i] = v.String()
	}
	idx := strings.Join(idxStrs, va.script.SubSep)

	// Associate the final argument with the index string.
	va.data[idx] = argVals[len(argVals)-1]
}

// Get returns the Value associated with a given index into a ValueArray.
// Multiple indexes can be specified to simulate multidimensional arrays.  (In
// fact, the indexes are concatenated into a single string with intervening
// Script.SubSep characters.)  The arguments can be provided either as Values
// or as any types that can be converted to Values.  If the index doesn't
// appear in the array, a zero value is returned.
func (va *ValueArray) Get(args ...interface{}) *Value {
	// Ensure we were given at least one index.
	if len(args) < 1 {
		panic("ValueArray.Get requires at least one index")
	}

	// Convert each argument to a Value.
	argVals := make([]*Value, len(args))
	for i, arg := range args {
		v, ok := arg.(*Value)
		if !ok {
			v = va.script.NewValue(arg)
		}
		argVals[i] = v
	}

	// Handle the most common case: a single index.
	if len(args) == 1 {
		vv, found := va.data[argVals[0].String()]
		if !found {
			return va.script.NewValue("")
		}
		return vv
	}

	// Merge the indexes into a single string.
	idxStrs := make([]string, len(argVals))
	for i, v := range argVals {
		idxStrs[i] = v.String()
	}
	idx := strings.Join(idxStrs, va.script.SubSep)

	// Look up the index in the associative array.
	vv, found := va.data[idx]
	if !found {
		return va.script.NewValue("")
	}
	return vv
}

// Delete deletes a key and associated value from a ValueArray.  Multiple
// indexes can be specified to simulate multidimensional arrays.  (In fact, the
// indexes are concatenated into a single string with intervening Script.SubSep
// characters.)  The arguments can be provided either as Values or as any types
// that can be converted to Values.  If no argument is provided, the entire
// ValueArray is emptied.
func (va *ValueArray) Delete(args ...interface{}) {
	// If we were given no arguments, delete the entire array.
	if args == nil {
		va.data = make(map[string]*Value)
		return
	}

	// Convert each argument to a Value.
	argVals := make([]*Value, len(args))
	for i, arg := range args {
		v, ok := arg.(*Value)
		if !ok {
			v = va.script.NewValue(arg)
		}
		argVals[i] = v
	}

	// Handle the most common case: a single index.
	if len(args) == 1 {
		delete(va.data, argVals[0].String())
		return
	}

	// Merge the indexes into a single string.
	idxStrs := make([]string, len(argVals))
	for i, v := range argVals {
		idxStrs[i] = v.String()
	}
	idx := strings.Join(idxStrs, va.script.SubSep)

	// Delete the index from the associative array.
	delete(va.data, idx)
}

// Keys returns all keys in the associative array in undefined order.
func (va *ValueArray) Keys() []*Value {
	keys := make([]*Value, 0, len(va.data))
	for kstr := range va.data {
		keys = append(keys, va.script.NewValue(kstr))
	}
	return keys
}

// Values returns all values in the associative array in undefined order.
func (va *ValueArray) Values() []*Value {
	vals := make([]*Value, 0, len(va.data))
	for _, v := range va.data {
		vals = append(vals, va.script.NewValue(v))
	}
	return vals
}
