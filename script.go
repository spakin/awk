// This file lets users define and execute AWK-like scripts within Go.

package awk

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

// A parseState indicates where we are in our parsing state.
type parseState int

// The following are the possibilities for a parseState.
const (
	atBegin  parseState = iota // Before any records are read
	inMiddle                   // While records are being read
	atEnd                      // After all records are read
)

// A Script contains all the internal state for an AWK-like script.
type Script struct {
	State   interface{} // Arbitrary, user-supplied data
	ConvFmt string      // Conversion format for numbers, "%.6g" by default
	NR      int         // Number of input records seen so far
	NF      int         // Number of fields in the current input record

	rs        string                    // Input record separator, newline by default
	fs        string                    // Input field separator, space by default
	rules     []statement               // List of pattern-action pairs to execute
	fields    []*Value                  // Fields in the current record; fields[0] is the entire record
	regexps   map[string]*regexp.Regexp // Map from a regular-expression string to a compiled regular expression
	rsScanner *bufio.Scanner            // Scanner associated with RS
	input     *bufio.Reader             // Script input stream
	parsing   parseState                // What we're currently parsing
}

// NewScript initializes a new Script with default values.
func NewScript() *Script {
	return &Script{
		ConvFmt: "%.6g",
		NR:      0,
		NF:      0,
		rs:      "\n",
		fs:      " ",
		fields:  make([]*Value, 0),
		rules:   make([]statement, 0, 10),
		regexps: make(map[string]*regexp.Regexp, 10),
	}
}

// SetRS sets the current input record separator (really, a record terminator).
func (s *Script) SetRS(rs string) {
	// Store the new record terminator.
	s.rs = rs

	// Create (and store) a new scanner based on the new record terminator.
	if s.input != nil {
		s.rsScanner = bufio.NewScanner(s.input)
		s.rsScanner.Split(s.makeRecordSplitter())
	}
}

// SetFS sets the current input field separator.
func (s *Script) SetFS(fs string) { s.fs = fs }

// F returns a specified field of the current record.  Field numbers are
// 1-based.  Field 0 refers to the entire record.  Requesting a field greater
// than NF returns a zero value.  Requesting a negative field number panics
// with an out-of-bounds error.
func (s *Script) F(i int) *Value {
	if i < len(s.fields) {
		return s.fields[i]
	}
	return s.NewValue("")
}

// SetF sets a field of the current record to the given Value.  Field numbers
// are 1-based.  Field 0 refers to the entire record.  Setting it causes the
// entire line to be reparsed (and NF recomputed).  Setting a field numbered
// larger than NF extends NF to that value.  Setting a negative field number
// panics with an out-of-bounds error.
func (s *Script) SetF(i int, v *Value) {
	// Zero index: Assign and reparse the entire record.
	if i == 0 {
		s.splitRecord(v.String())
		return
	}

	// Index larger than NF: extend NF and try again.
	for i >= len(s.fields) {
		s.fields = append(s.fields, s.NewValue(""))
	}

	// Index not larger than (the possibly modified) NF: write the field.
	s.fields[i] = v
}

// A PatternFunc represents a pattern to match against.  It is expected to
// examine the state in the given Script then return either true or false.  If
// it returns true, the corresponding ActionFunc is executed.  Otherwise, the
// corresponding ActionFunc is not executed.
type PatternFunc func(*Script) bool

// An ActionFunc represents an action to perform when the corresponding
// PatternFunc returns true.  It can be arbitrary Go code, such as to write
// output to a file.
type ActionFunc func(*Script)

// A statement represents a single pattern-action pair.
type statement struct {
	Pattern PatternFunc
	Action  ActionFunc
}

// The Begin pattern is true at the beginning of a script, before any records
// have been read.
func Begin(s *Script) bool {
	return s.parsing == atBegin
}

// The matchAny pattern is true in the middle of a script, when a record is
// available for parsing.
func matchAny(s *Script) bool {
	return s.parsing == inMiddle
}

// The End pattern is true at the end of a script, after all records have been
// read.
func End(s *Script) bool {
	return s.parsing == atEnd
}

// The printRecord statement outputs the current record verbatim to the
// standard output device.
func printRecord(s *Script) {
	fmt.Printf("%v\n", s.fields[0])
}

// AppendStmt appends a pattern-action pair to a Script.
func (s *Script) AppendStmt(p PatternFunc, a ActionFunc) {
	stmt := statement{
		Pattern: p,
		Action:  a,
	}
	if p == nil {
		stmt.Pattern = matchAny
	}
	if a == nil {
		stmt.Action = printRecord
	}
	s.rules = append(s.rules, stmt)
}

// makeFieldSplitter returns a splitter that returns the next field.
func (s *Script) makeFieldSplitter() func([]byte, bool) (int, []byte, error) {
	// Separator is empty: return the next rune.
	if s.fs == "" {
		return bufio.ScanRunes
	}

	// Separator is a single space: return the next word.
	if s.fs == " " {
		return bufio.ScanWords
	}

	// Separator is a single character: scan based on that.  This code is
	// derived from the bufio.ScanWords source.
	if utf8.RuneCountInString(s.fs) == 1 {
		// Ensure the separator character is valid.
		firstRune, _ := utf8.DecodeRuneInString(s.fs)
		if firstRune == utf8.RuneError {
			return func(data []byte, atEOF bool) (int, []byte, error) {
				return 0, nil, errors.New("Invalid rune in separator")
			}
		}

		// The separator is valid.  Return a splitter customized to
		// that separator.
		returnedFinalToken := false // true=already returned a final, non-terminated token; false=didn't
		return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			// Scan until we see a separator or run out of data.
			for width, i := 0, 0; i < len(data); i += width {
				var r rune
				r, width = utf8.DecodeRune(data[i:])
				if r == utf8.RuneError {
					return 0, nil, errors.New("Invalid rune in input data")
				}
				if r == firstRune {
					return i + width, data[:i], nil
				}
			}

			// We didn't see a separator.  If we're at EOF, we have
			// a final, non-terminated token.  Return it (unless we
			// already did).
			if atEOF && !returnedFinalToken {
				returnedFinalToken = true
				return len(data), data, nil
			}

			// Request more data.
			return 0, nil, nil
		}
	}

	// Separator is multiple characters: treat it as a regular expression,
	// and scan based on that.  This code is also derived from the
	// bufio.ScanWords source.
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// BUG(pakin): Multiple-character separators are not yet implemented.
		return 0, nil, errors.New("Multiple-character separators are not yet implemented")
	}
}

// makeRecordSplitter returns a splitter that returns the next record.
// Although all the AWK documentation I've read define RS as a record
// separator, as far as I can tell, AWK in fact treats it as a record
// *terminator* so we do, too.
func (s *Script) makeRecordSplitter() func([]byte, bool) (int, []byte, error) {
	// Terminator is empty: return the next rune.
	if s.rs == "" {
		return bufio.ScanRunes
	}

	// Terminator is a single space: return the next word.
	if s.rs == " " {
		return bufio.ScanWords
	}

	// Terminator is a single character: scan based on that.  This code is
	// derived from the bufio.ScanWords source.
	if utf8.RuneCountInString(s.rs) == 1 {
		// Ensure the terminator character is valid.
		firstRune, _ := utf8.DecodeRuneInString(s.rs)
		if firstRune == utf8.RuneError {
			return func(data []byte, atEOF bool) (int, []byte, error) {
				return 0, nil, errors.New("Invalid rune in terminator")
			}
		}

		// The terminator is valid.  Return a splitter customized to
		// that terminator.
		return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			// Scan until we see a terminator or run out of data.
			for width, i := 0, 0; i < len(data); i += width {
				var r rune
				r, width = utf8.DecodeRune(data[i:])
				if r == utf8.RuneError {
					return 0, nil, errors.New("Invalid rune in input data")
				}
				if r == firstRune {
					return i + width, data[:i], nil
				}
			}

			// We didn't see a terminator.  If we're at EOF, we
			// have a final, non-terminated token.  Return it if
			// it's nonempty.
			if atEOF && len(data) > 0 {
				return len(data), data, nil
			}

			// Request more data.
			return 0, nil, nil
		}
	}

	// Terminator is multiple characters: treat it as a regular expression,
	// and scan based on that.  This code is also derived from the
	// bufio.ScanWords source.
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// BUG(pakin): Multiple-character terminators are not yet implemented.
		return 0, nil, errors.New("Multiple-character separators are not yet implemented")
	}
}

// Read the next record from a stream and return it.
func (s *Script) readRecord() (string, error) {
	// Return the next record.
	if s.rsScanner.Scan() {
		return s.rsScanner.Text(), nil
	}
	if err := s.rsScanner.Err(); err != nil {
		return "", err
	} else {
		return "", io.EOF
	}
}

// Split a record into fields.  Store the fields in the Script struct's F field.
// As in real AWK, field 0 is the entire record.
func (s *Script) splitRecord(rec string) error {
	fsScanner := bufio.NewScanner(strings.NewReader(rec))
	fsScanner.Split(s.makeFieldSplitter())
	fields := make([]*Value, 0, 100)
	fields = append(fields, s.NewValue(rec))
	for fsScanner.Scan() {
		fields = append(fields, s.NewValue(fsScanner.Text()))
	}
	if err := fsScanner.Err(); err != nil {
		return err
	}
	s.fields = fields
	return nil
}

// Execute a script against a given input stream.
func (s *Script) Run(r io.Reader) error {
	// Define a helper function that makes a pass through all user-defined
	// statements.
	walkStatements := func() {
		for _, rule := range s.rules {
			if rule.Pattern(s) {
				rule.Action(s)
			}
		}
	}

	// Wrap a buffered reader around the given reader.
	rb, ok := r.(*bufio.Reader)
	if !ok {
		rb = bufio.NewReader(r)
	}
	s.input = rb

	// Reinitialize most of our state.
	s.ConvFmt = "%.6g"
	s.NF = 0
	s.NR = 0
	s.SetFS(" ")
	s.SetRS("\n")

	// Process all Begin actions.
	s.parsing = atBegin
	walkStatements()

	// Process each record in turn.
	s.parsing = inMiddle
	for {
		// Read a record.
		rec, err := s.readRecord()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		s.NR++

		// Split the record into its constituent fields.
		s.splitRecord(rec)

		// Process all applicable actions.
		walkStatements()
	}

	// Process all End actions
	s.parsing = atEnd
	walkStatements()
	return nil
}
