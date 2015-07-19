// This file lets users define and execute AWK-like scripts within Go.

package awk

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"unicode/utf8"
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

// Execute a script against a given input stream.
func (s *Script) Run(r io.Reader) error {
	// Wrap a buffered reader around the given reader.
	rb, ok := r.(*bufio.Reader)
	if !ok {
		rb = bufio.NewReader(r)
	}
	s.input = rb

	// Process each record in turn.
	for {
		// Read a record.
		record, err := s.readRecord()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		_ = record // Temporary
	}
	return nil
}
