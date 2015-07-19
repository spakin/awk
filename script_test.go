// This file tests script primitives.

package awk

import (
	"bufio"
	"strings"
	"testing"
)

// TestReadRecordNewline tests reading newline-separated records.
func TestReadRecordNewline(t *testing.T) {
	// Test with no trailing newline.
	allRecords := []string{"Word", "Foo bar baz quux", "More text"}
	scr := NewScript()
	allRecordsStr := strings.Join(allRecords, "\n")
	scr.input = bufio.NewReader(strings.NewReader(allRecordsStr))
	for _, oneRecord := range allRecords {
		rec, err := scr.readRecord()
		if err != nil {
			t.Fatal(err)
		}
		if rec != oneRecord {
			t.Fatalf("Expected %q but received %q", oneRecord, rec)
		}
	}

	// Test with a trailing newline.
	allRecordsStr += "\n"
	scr.input = bufio.NewReader(strings.NewReader(allRecordsStr))
	delete(scr.rsScanner, scr.RS)
	for _, oneRecord := range allRecords {
		rec, err := scr.readRecord()
		if err != nil {
			t.Fatal(err)
		}
		if rec != oneRecord {
			t.Fatalf("Expected %q but received %q", oneRecord, rec)
		}
	}
}
