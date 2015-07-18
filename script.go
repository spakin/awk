// This file lets users define and execute AWK-like scripts within Go.

package awk

import (
	"regexp"
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

	rules   []Statement               // List of pattern-action pairs to execute
	regexps map[string]*regexp.Regexp // Map from a regular-expression string to a compiled regular expression
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
