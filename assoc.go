// This file defines an AWK-like associative array, ValueArray.

package awk

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

// Set assigns a Value to an index of a ValueArray.  The arguments can be
// provided either as Values or as any types that can be converted to Values.
func (va *ValueArray) Set(idx, val interface{}) {
	// Convert the index and value to Values.
	vi, ok := idx.(*Value)
	if !ok {
		vi = va.script.NewValue(idx)
	}
	vv, ok := val.(*Value)
	if !ok {
		vv = va.script.NewValue(val)
	}

	// Map using the string version of the index rather than the Value
	// pointer.  Otherwise, two Values with the same contents would be
	// treated as different.
	va.data[vi.String()] = vv
}

// Get returns the Value associated with a given index into a ValueArray.  The
// argument can be provided either as a Value or as any type that can be
// converted to a Value.  If the index doesn't appear in the array, a zero
// value is returned.
func (va *ValueArray) Get(idx interface{}) *Value {
	// Convert the index to a Value.
	vi, ok := idx.(*Value)
	if !ok {
		vi = va.script.NewValue(idx)
	}

	// Map using the string version of the index rather than the Value
	// pointer.  Otherwise, two Values with the same contents would be
	// treated as different.
	vv, found := va.data[vi.String()]
	if !found {
		return va.script.NewValue("")
	}
	return vv
}
