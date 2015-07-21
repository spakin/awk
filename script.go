// This file lets users define and execute AWK-like scripts within Go.

package awk

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
	"fmt"
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
	FS      string      // Input field separator, space by default
	NF      int         // Number of fields in the current input record
	NR      int         // Number of input records seen so far
	RS      string      // Input record separator, newline by default
	F       []*Value    // Fields in the current record; F[0] is the entire record

	rules     []Statement               // List of pattern-action pairs to execute
	regexps   map[string]*regexp.Regexp // Map from a regular-expression string to a compiled regular expression
	rsScanner *bufio.Scanner            // Scanner associated with RS
	prevRS    string                    // RS associated with rsScanner
	input     *bufio.Reader             // Script input stream
	parsing   parseState                // What we're currently parsing
}

// NewScript initializes a new Script with default values.
func NewScript() *Script {
	return &Script{
		ConvFmt: "%.6g",
		FS:      " ",
		NF:      0,
		NR:      0,
		RS:      "\n",
		rules:   make([]Statement, 0, 10),
		regexps: make(map[string]*regexp.Regexp, 10),
	}
}

// A Statement represents a single pattern-action pair.
type Statement struct {
	Pattern func(*Script) bool // true: run Action; false: go to next statement
	Action  func(*Script)      // Operations to perform when Pattern returns true
}

// The Begin pattern is true at the beginning of a script, before any records
// have been read.
func Begin(s *Script) bool {
	return s.parsing == atBegin
}

// The End pattern is true at the end of a script, after all records have been
// read.
func End(s *Script) bool {
	return s.parsing == atEnd
}

// The Print statement outputs the current record verbatim to the standard
// output device.
func Print(s *Script) {
	fmt.Printf("%v%s", s.F[0], s.RS)
}

// AppendStmt appends a pattern-action pair to a Script.
func (s *Script) AppendStmt(st Statement) {
	s.rules = append(s.rules, st)
}

// Return a splitter that can split the next record or field.
func (s *Script) makeSplitter(sep string) func([]byte, bool) (int, []byte, error) {
	// Separator is empty: return the next rune.
	if sep == "" {
		return bufio.ScanRunes
	}

	// Separator is a single space: return the next word.
	if sep == " " {
		return bufio.ScanWords
	}

	// Separator is a single character: scan based on that.  This code is
	// derived from the bufio.ScanWords source.
	if utf8.RuneCountInString(sep) == 1 {
		firstRune, _ := utf8.DecodeRuneInString(sep)
		return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			// Ensure the separator character is valid.
			if firstRune == utf8.RuneError {
				return 0, nil, errors.New("Invalid rune in separator")
			}

			// Scan until we see a separator.
			for width, i := 0, 0; i < len(data); i += width {
				var r rune
				r, width = utf8.DecodeRune(data[i:])
				if r == utf8.RuneError {
					return 0, nil, errors.New("Invalid rune in input data")
				}
				if r == firstRune {
					return i + width, data[0:i], nil
				}
			}

			// If we're at EOF, we have a final, non-empty,
			// non-terminated token. Return it.
			if atEOF && len(data) > 0 {
				return len(data), data[0:], nil
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

// Read the next record from a stream and return it.
func (s *Script) readRecord() (string, error) {
	// Reuse the existing scanner if RS hasn't changed.  Otherwise, create
	// (and store) a new scanner.
	if s.rsScanner == nil || s.RS != s.prevRS {
		// Create a new scanner.
		s.rsScanner = bufio.NewScanner(s.input)
		s.rsScanner.Split(s.makeSplitter(s.RS))
		s.prevRS = s.RS
	}

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
func (s *Script) splitRecord(rec string) error {
	fsScanner := bufio.NewScanner(strings.NewReader(rec))
	fsScanner.Split(s.makeSplitter(s.FS))
	fields := make([]*Value, 0, 100)
	for fsScanner.Scan() {
		fields = append(fields, s.NewValue(fsScanner.Text()))
	}
	if err := fsScanner.Err(); err != nil {
		return err
	}
	s.F = fields
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
