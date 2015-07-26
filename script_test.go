// This file tests script primitives.

package awk

import (
	"bufio"
	"regexp"
	"strings"
	"testing"
)

// TestReadRecordNewline tests reading newline-separated records.
func TestReadRecordNewline(t *testing.T) {
	// Define the basic test we plan to repeat.
	allRecords := []string{"X", "Word", "More than one word", "", "More text"}
	allRecordsStr := strings.Join(allRecords, "\n")
	scr := NewScript()
	doTest := func() {
		scr.input = bufio.NewReader(strings.NewReader(allRecordsStr))
		scr.SetRS("\n")
		scr.rsScanner = bufio.NewScanner(scr.input)
		scr.rsScanner.Split(scr.makeRecordSplitter())
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

	// Test with no trailing newline.
	doTest()

	// Test with a trailing newline.
	allRecordsStr += "\n"
	doTest()
}

// TestReadRecordWhitespace tests reading whitespace-separated records.
func TestReadRecordWhitespace(t *testing.T) {
	allRecordsStr := "  banana banana banana  banana   banana banana\tbanana banana\nbanana banana"
	want := []string{
		"",
		"",
		"banana",
		"banana",
		"banana",
		"",
		"banana",
		"",
		"",
		"banana",
		"banana\tbanana",
		"banana\nbanana",
		"banana",
	}
	scr := NewScript()
	scr.input = bufio.NewReader(strings.NewReader(allRecordsStr))
	scr.SetRS(" ")
	scr.rsScanner = bufio.NewScanner(scr.input)
	scr.rsScanner.Split(scr.makeRecordSplitter())
	for _, str := range want {
		rec, err := scr.readRecord()
		if err != nil {
			t.Fatal(err)
		}
		if rec != str {
			t.Fatalf("Expected %q but received %q", str, rec)
		}
	}
}

// TestReadRecordRE tests reading regular-expression-separated records.
func TestReadRecordRE(t *testing.T) {
	allRecordsStr := "hello<foo>howdy</foo>hello<bar>yellow</bar>hello<baz>goodbye</baz>"
	scr := NewScript()
	scr.input = bufio.NewReader(strings.NewReader(allRecordsStr))
	scr.SetRS(`<[^>]+>[^<]*<[^>]+>`)
	scr.rsScanner = bufio.NewScanner(scr.input)
	scr.rsScanner.Split(scr.makeRecordSplitter())
	for i := 0; i < 3; i++ {
		rec, err := scr.readRecord()
		if err != nil {
			t.Fatal(err)
		}
		if rec != "hello" {
			t.Fatalf("Expected %q but received %q", "hello", rec)
		}
	}
}

// TestSplitRecordWhitespace tests splitting a record into
// whitespace-separated fields.
func TestSplitRecordWhitespace(t *testing.T) {
	recordStr := "The woods are lovely,  dark and    deep,"
	fields := regexp.MustCompile(`\s+`).Split(recordStr, -1)
	scr := NewScript()
	scr.splitRecord(recordStr)
	for i, f := range fields {
		if scr.F(i+1).String() != f {
			t.Fatalf("Expected %q but received %q", f, scr.F(i+1))
		}
	}
}

// TestSplitRecordComma tests splitting a record into comma-separated fields.
func TestSplitRecordComma(t *testing.T) {
	recordStr := "The woods are lovely,  dark and    deep,"
	fields := strings.Split(recordStr, ",")
	scr := NewScript()
	scr.SetFS(",")
	scr.splitRecord(recordStr)
	for i, f := range fields {
		if scr.F(i+1).String() != f {
			t.Fatalf("Expected %q but received %q", f, scr.F(i+1))
		}
	}
}

// TestSplitFieldRE tests splitting a field based on a regular expression.
func TestSplitFieldRE(t *testing.T) {
	// Determine what we want to provide and see in return.
	recordStr := "foo-bar---baz------------quux--corge-grault---garply-"
	re, err := regexp.Compile(`\w+`)
	if err != nil {
		t.Fatal(err)
	}
	words := re.FindAllString(recordStr, -1)
	words = append(words, "")

	// Split the record.
	scr := NewScript()
	scr.SetFS("-+")
	scr.splitRecord(recordStr)

	// Check the result.
	for i := 1; i <= scr.NF; i++ {
		f := scr.F(i).String()
		if f != words[i-1] {
			t.Fatalf("Expected %q for field %d but received %q", words[i-1], i, f)
		}
	}
}

// TestSplitFieldREIgnCase tests splitting a field based on a case-insensitive
// regular expression.
func TestSplitFieldREIgnCase(t *testing.T) {
	// Determine what we want to provide and see in return.
	recordStr := "fooxbarXxxbazxxXXxxxXxxXxquucksxXcorgexgraultxxxgarplyx"
	re, err := regexp.Compile(`[fobarzqucksgeltpy]+`)
	if err != nil {
		t.Fatal(err)
	}
	words := re.FindAllString(recordStr, -1)
	words = append(words, "")

	// Split the record.
	scr := NewScript()
	scr.SetFS("x+")
	scr.IgnoreCase(true)
	scr.splitRecord(recordStr)

	// Check the result.
	for i := 1; i <= scr.NF; i++ {
		f := scr.F(i).String()
		if f != words[i-1] {
			t.Fatalf("Expected %q for field %d but received %q", words[i-1], i, f)
		}
	}
}

// TestBeginEnd tests creating and running a script that contains a Begin
// action and an End action.
func TestBeginEnd(t *testing.T) {
	scr := NewScript()
	val := 123
	scr.AppendStmt(Begin, func(s *Script) { val *= 10 })
	scr.AppendStmt(End, func(s *Script) { val += 4 })
	err := scr.Run(strings.NewReader("dummy data"))
	if err != nil {
		t.Fatal(err)
	}
	if val != 1234 {
		t.Fatalf("Expected 1234 but received %d", val)
	}
}

// TestSimpleSum tests adding up a column of numbers.
func TestSimpleSum(t *testing.T) {
	scr := NewScript()
	sum := 0
	scr.AppendStmt(nil, func(s *Script) { sum += s.F(1).Int() })
	err := scr.Run(strings.NewReader("2\n4\n6\n8\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 20 {
		t.Fatalf("Expected 20 but received %d", sum)
	}
}

// TestRunTwice tests running the same script twice.
func TestRunTwice(t *testing.T) {
	// Run once.
	scr := NewScript()
	sum := 0
	scr.AppendStmt(nil, func(s *Script) { sum += s.F(1).Int() * s.NR })
	err := scr.Run(strings.NewReader("1\n3\n5\n7\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 50 {
		t.Fatalf("Expected 50 but received %d on the first trial", sum)
	}

	// Run again.
	sum = 0
	err = scr.Run(strings.NewReader("1\n3\n5\n7\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 50 {
		t.Fatalf("Expected 50 but received %d on the second trial", sum)
	}
}

// TestFieldCreation tests creating ("autovivifying" in Perl-speak) new fields.
func TestFieldCreation(t *testing.T) {
	scr := NewScript()
	sum := 0
	scr.AppendStmt(nil, func(s *Script) { sum += 1 << uint(s.F(2).Int()) })
	err := scr.Run(strings.NewReader("x 3\ny 2\n\nz 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 15 {
		t.Fatalf("Expected 15 but received %d", sum)
	}
}

// TestRecordReplacement tests overwriting field 0 with a new record.
func TestRecordReplacement(t *testing.T) {
	scr := NewScript()
	sum := 0
	scr.AppendStmt(nil, func(s *Script) {
		sum += s.F(2).Int()
		s.SetF(0, s.NewValue("10 20 30 40 50"))
		sum += s.F(5).Int()
	})
	err := scr.Run(strings.NewReader("x 3\ny 2\n\nz 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 206 {
		t.Fatalf("Expected 206 but received %d", sum)
	}
}

// TestRecordChangeCase tests changing IgnoreCase during the execution of a
// script.
func TestRecordChangeCase(t *testing.T) {
	scr := NewScript()
	sum := 0
	scr.AppendStmt(func(s *Script) bool { return s.F(1).Int()%2 == 0 },
		func(s *Script) { sum += s.F(1).Int() })
	scr.AppendStmt(func(s *Script) bool { return s.NR == 3 },
		func(s *Script) { s.IgnoreCase(true) })
	scr.SetRS("EOL")
	err := scr.Run(strings.NewReader("1EOL2EOL3EOL4Eol5eol6eoL"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 12 {
		t.Fatalf("Expected 12 but received %d", sum)
	}
}

// TestRecordBlankLines tests the AWK special case of blank-line-separated
// records.
func TestRecordBlankLines(t *testing.T) {
	recordStr := "uno\ndos\n\ntres\ncuatro\n\ncinco,seis,siete\nocho\n\nnueve,diez\n\n"
	expected := regexp.MustCompile(`[\n,]+`).Split(recordStr, -1)
	expected = expected[:len(expected)-1] // Skip empty final record.
	actual := make([]string, 0, 10)
	scr := NewScript()
	scr.SetRS("")
	scr.SetFS(",")
	scr.AppendStmt(nil, func(s *Script) {
		for i := 1; i <= s.NF; i++ {
			actual = append(actual, s.F(i).String())
		}
	})
	err := scr.Run(strings.NewReader(recordStr))
	if err != nil {
		t.Fatal(err)
	}
	for i, s1 := range expected {
		s2 := actual[i]
		if s1 != s2 {
			t.Fatalf("Expected %v but received %v", expected, actual)
		}
	}
}

// TestExit tests premature script termination.
func TestExit(t *testing.T) {
	scr := NewScript()
	sum := 0
	scr.AppendStmt(Begin, func(s *Script) { s.IgnoreCase(true) })
	scr.AppendStmt(nil, func(s *Script) { sum += s.F(1).Int() })
	scr.AppendStmt(func(s *Script) bool { return s.F(1).StrEqual("stop") },
		func(s *Script) { s.Exit() })
	err := scr.Run(strings.NewReader("111\n222\n333\n444\nSTOP\n555\n666\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 1110 {
		t.Fatalf("Expected 1110 but received %d", sum)
	}
}

// TestRecordRange tests range patterns.
func TestRecordRange(t *testing.T) {
	scr := NewScript()
	all := []string{
		"bad",
		"terrible",
		"BEGIN",
		"good",
		"great",
		"fantastic",
		"END",
		"awful",
		"dreadful",
	}
	want := []string{
		"BEGIN",
		"good",
		"great",
		"fantastic",
		"END",
	}
	got := make([]string, 0, 10)
	scr.AppendStmt(Range(func(s *Script) bool { return s.F(1).Match("BEGIN") },
		func(s *Script) bool { return s.F(1).Match("END") }),
		func(s *Script) { got = append(got, s.F(1).String()) })
	err := scr.Run(strings.NewReader(strings.Join(all, "\n")))
	if err != nil {
		t.Fatal(err)
	}
	for i, s1 := range want {
		s2 := got[i]
		if s1 != s2 {
			t.Fatalf("Expected %q but received %q", s1, s2)
		}
	}
}

// TestSplitRecordRE tests splitting record into regexp-separated fields.
func TestSplitRecordRE(t *testing.T) {
	scr := NewScript()
	pluses := 0
	scr.AppendStmt(Begin, func(s *Script) { s.SetRS(`\++`) })
	scr.AppendStmt(nil, func(s *Script) { pluses += len(s.RT) })
	err := scr.Run(strings.NewReader("a++++++a++a++++a+++a+++++a+"))
	if err != nil {
		t.Fatal(err)
	}
	if pluses != 21 {
		t.Fatalf("Expected 21 but received %d", pluses)
	}
}
