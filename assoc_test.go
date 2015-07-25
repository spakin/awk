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

// TestStringStringArray tests Get/Set operations on an associative array that
// maps strings to strings.
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

// TestMultiDimArray tests Get/Set operations on a "multidimensional"
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

// TestArrayKeys tests the Keys operation on an associative array.
func TestArrayKeys(t *testing.T) {
	scr := NewScript()
	a := scr.NewValueArray()
	for i := 10; i <= 100; i += 10 {
		a.Set(i, i*2)
	}
	ksum := 0
	for _, k := range a.Keys() {
		ksum += k.Int()
	}
	if ksum != 550 {
		t.Fatalf("Expected 550 but received %d", ksum)
	}
}

// TestArrayValues tests the Values operation on an associative array.
func TestArrayValues(t *testing.T) {
	scr := NewScript()
	a := scr.NewValueArray()
	for i := 10; i <= 100; i += 10 {
		a.Set(i, i*2)
	}
	vsum := 0
	for _, v := range a.Values() {
		vsum += v.Int()
	}
	if vsum != 1100 {
		t.Fatalf("Expected 1100 but received %d", vsum)
	}
}
