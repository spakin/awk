// This file tests operations on associative arrays

package awk

import (
	"testing"
)

// TestIntIntArray tests Get/Set operations on an associative array that
// maps integers to integers.
func TestIntIntArray(t *testing.T) {
	scr := NewScript()
	a := scr.NewValueArray()
	for i := 0; i < 10; i++ {
		a.Set(i, i*10)
	}
	for i := 9; i >= 0; i-- {
		got := a.Get(i).Int()
		if got != i*10 {
			t.Fatalf("Expected %d but received %d", i*10, got)
		}
	}
}

// TestValueValueArray tests Get/Set operations on an
// associative array that maps Values to Values.
func TestValueValueArray(t *testing.T) {
	scr := NewScript()
	a := scr.NewValueArray()
	for i := 0; i < 10; i++ {
		a.Set(scr.NewValue(i), scr.NewValue(i*10))
	}
	for i := 9; i >= 0; i-- {
		got := a.Get(scr.NewValue(i)).Int()
		if got != i*10 {
			t.Fatalf("Expected %d but received %d", i*10, got)
		}
	}
}

// TestStringStringArray tests basic Get/Set operations on an associative array
// that maps strings to strings.
func TestStringStringArray(t *testing.T) {
	scr := NewScript()
	a := scr.NewValueArray()
	keys := []string{"The", "tree", "has", "entered", "my", "hands"}
	values := []string{"The", "sap", "has", "ascended", "my", "arms"}
	for i, k := range keys {
		a.Set(k, values[i])
	}
	for i, k := range keys {
		want := values[i]
		got := a.Get(k).String()
		if got != want {
			t.Fatalf("Expected %q but received %q", want, got)
		}
	}
}

// TestMultiDimArray tests basic Get/Set operations on a "multidimensional"
// associative array.
func TestMultiDimArray(t *testing.T) {
	scr := NewScript()
	a := scr.NewValueArray()
	for i := 9; i >= 0; i-- {
		for j := 9; j >= 0; j-- {
			a.Set(i, j, i*10+j)
		}
	}
	for i := 0; i < 10; i++ {
		for j := 9; j >= 0; j-- {
			got := a.Get(i, j).Int()
			if got != i*10+j {
				t.Fatalf("Expected %d but received %d", i*10+j, got)
			}
		}
	}
}
