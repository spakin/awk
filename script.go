// This file lets users define and execute AWK-like scripts within Go.

package awk

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

// A parseState indicates where we are in our parsing state.
type parseState int

// The following are the possibilities for a parseState.
const (
	notRunning parseState = iota // Before/after Run was called
	atBegin                      // Before any records are read
	inMiddle                     // While records are being read
	atEnd                        // After all records are read
)

// A stopState describes premature stop conditions.
type stopState int

// The following are possibilities for a stopState.
const (
	dontStop   stopState = iota // Normal execution
	stopRec                     // Abort the current record
	stopScript                  // Abort the entire script
)

// A Script contains all the internal state for an AWK-like script.
type Script struct {
	State   interface{} // Arbitrary, user-supplied data
	Output  io.Writer   // Output stream (defaults to os.Stdout)
	ConvFmt string      // Conversion format for numbers, "%.6g" by default
	SubSep  string      // Separator for simulated multidimensional arrays
	NR      int         // Number of input records seen so far
	NF      int         // Number of fields in the current input record
	RT      string      // Actual string terminating the current record
	RStart  int         // 1-based index of the previous regexp match (Value.Match)
	RLength int         // Length of the previous regexp match (Value.Match)

	rs        string                    // Input record separator, newline by default
	fs        string                    // Input field separator, space by default
	ors       string                    // Output record separator, newline by default
	ofs       string                    // Output field separator, space by default
	ignCase   bool                      // true: REs are case-insensitive; false: case-sensitive
	rules     []statement               // List of pattern-action pairs to execute
	fields    []*Value                  // Fields in the current record; fields[0] is the entire record
	regexps   map[string]*regexp.Regexp // Map from a regular-expression string to a compiled regular expression
	rsScanner *bufio.Scanner            // Scanner associated with RS
	input     *bufio.Reader             // Script input stream
	state     parseState                // What we're currently parsing
	stop      stopState                 // What we should stop doing
}

// NewScript initializes a new Script with default values.
func NewScript() *Script {
	return &Script{
		Output:  os.Stdout,
		ConvFmt: "%.6g",
		SubSep:  "\034",
		NR:      0,
		NF:      0,
		rs:      "\n",
		fs:      " ",
		ors:     "\n",
		ofs:     " ",
		ignCase: false,
		rules:   make([]statement, 0, 10),
		fields:  make([]*Value, 0),
		regexps: make(map[string]*regexp.Regexp, 10),
		state:   notRunning,
	}
}

// SetRS sets the input record separator (really, a record terminator).  In the
// current implementation, it will panic if called after the first record is
// read.  (It is acceptable to call SetRS from a Begin action, though.)  As in
// AWK, if the record separator is a single character, that character is used
// to separate records; if the record separator is multiple characters, it's
// treated as a regular expression (subject to the current setting of
// Script.IgnoreCase); and if the record separator is an empty string, records
// are separated by blank lines.  That last case implicitly causes newlines to
// be accepted as a field separator in addition to whatever was specified by
// SetFS.
func (s *Script) SetRS(rs string) {
	if s.state == inMiddle {
		panic("SetRS was called from a running script")
	}
	s.rs = rs
}

// SetFS sets the input field separator.  As in AWK, if the field separator is
// a single space (the default), fields are separated by runs of whitespace; if
// the field separator is any other single character, that character is used to
// separate fields; if the field separator is an empty string, each individual
// character becomes a separate field; and if the field separator is multiple
// characters, it's treated as a regular expression (subject to the current
// setting of Script.IgnoreCase).
func (s *Script) SetFS(fs string) { s.fs = fs }

// SetORS sets the output record separator.
func (s *Script) SetORS(ors string) { s.ors = ors }

// SetOFS sets the output field separator.
func (s *Script) SetOFS(ofs string) { s.ofs = ofs }

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
	if i >= len(s.fields) {
		for i >= len(s.fields) {
			s.fields = append(s.fields, s.NewValue(""))
		}
		s.NF = len(s.fields) - 1
	}

	// Index not larger than (the possibly modified) NF: write the field.
	s.fields[i] = v
}

// IgnoreCase specifies whether regular-expression and string comparisons
// should be performed in a case-insensitive manner.
func (s *Script) IgnoreCase(ign bool) {
	s.ignCase = ign
}

// Println is like fmt.Println but honors the current output stream, output
// field separator, and output record separator.  If called with no arguments,
// Println outputs all fields in the current record.
func (s *Script) Println(args ...interface{}) {
	// No arguments: Output all fields of the current record.
	if args == nil {
		for i := 1; i <= s.NF; i++ {
			fmt.Fprintf(s.Output, "%v", s.F(i))
			if i == s.NF {
				fmt.Fprintf(s.Output, "%s", s.ors)
			} else {
				fmt.Fprintf(s.Output, "%s", s.ofs)
			}
		}
		return
	}

	// One or more arguments: Output them.
	for i, arg := range args {
		fmt.Fprintf(s.Output, "%v", arg)
		if i == len(args)-1 {
			fmt.Fprintf(s.Output, "%s", s.ors)
		} else {
			fmt.Fprintf(s.Output, "%s", s.ofs)
		}
	}
}

// A PatternFunc represents a pattern to match against.  It is expected to
// examine the state of the given Script then return either true or false.  If
// it returns true, the corresponding ActionFunc is executed.  Otherwise, the
// corresponding ActionFunc is not executed.
type PatternFunc func(*Script) bool

// An ActionFunc represents an action to perform when the corresponding
// PatternFunc returns true.
type ActionFunc func(*Script)

// A statement represents a single pattern-action pair.
type statement struct {
	Pattern PatternFunc
	Action  ActionFunc
}

// The Begin pattern is true only at the beginning of a script, before any
// records have been read.
func Begin(s *Script) bool {
	return s.state == atBegin
}

// The matchAny pattern is true only in the middle of a script, when a record
// is available for parsing.
func matchAny(s *Script) bool {
	return s.state == inMiddle
}

// The End pattern is true only at the end of a script, after all records have
// been read.
func End(s *Script) bool {
	return s.state == atEnd
}

// The printRecord statement outputs the current record verbatim to the current
// output stream.
func printRecord(s *Script) {
	fmt.Fprintf(s.Output, "%v%s", s.fields[0], s.ors)
}

// Next stops processing the current record and proceeds with the next record.
func (s *Script) Next() {
	if s.stop == dontStop {
		s.stop = stopRec
	}
}

// Exit stops processing the entire script, causing the Run method to return.
func (s *Script) Exit() {
	if s.stop == dontStop {
		s.stop = stopScript
	}
}

// Range combines two patterns into a single pattern that statefully returns
// true between the time the first and second pattern become true (both
// inclusively).
func Range(p1, p2 PatternFunc) PatternFunc {
	inRange := false
	return func(s *Script) bool {
		if inRange {
			inRange = !p2(s)
			return true
		} else {
			inRange = p1(s)
			return inRange
		}
	}
}

// AppendStmt appends a pattern-action pair to a Script.  If the pattern
// function is nil, the action will be performed on every record.  If the
// action function is nil, the record will be output verbatim to the standard
// output device.  The predefined pattern awk.Begin matches before any records
// have been read, and the predefined pattern awk.End matches after all records
// have been read.
func (s *Script) AppendStmt(p PatternFunc, a ActionFunc) {
	// Panic if we were called on a running script.
	if s.state != notRunning {
		panic("AppendStmt was called from a running script")
	}

	// Append a statement to the list of rules.
	stmt := statement{
		Pattern: p,
		Action:  a,
	}
	if p == nil {
		stmt.Pattern = matchAny
	} else {
		// Go unfortunately doesn't allow function comparisons.  Hence,
		// we resort to some trickery to determine if we were passed
		// Begin, End, or neither.  This trick does demand that pattern
		// execution be idempotent.
		s.state = atBegin
		isBegin := p(s)
		s.state = inMiddle
		isMiddle := p(s)
		s.state = atEnd
		isEnd := p(s)
		s.state = notRunning
		switch {
		case isBegin && !isMiddle && !isEnd:
		case !isBegin && !isMiddle && isEnd:
		default:
			stmt.Pattern = func(s *Script) bool {
				if s.state == inMiddle {
					return p(s)
				}
				return false
			}
		}
	}
	if a == nil {
		stmt.Action = printRecord
	}
	s.rules = append(s.rules, stmt)
}

// compileRegexp caches and returns the result of regexp.Compile.  It
// automatically prepends "(?i)" to the expression if the script is currently
// set to perform case-insensitive regular-expression matching.
func (s *Script) compileRegexp(expr string) (*regexp.Regexp, error) {
	if s.ignCase {
		expr = "(?i)" + expr
	}
	re, found := s.regexps[expr]
	if found {
		return re, nil
	}
	var err error
	re, err = regexp.Compile(expr)
	if err != nil {
		return nil, err
	}
	s.regexps[expr] = re
	return re, nil
}

// makeFieldSplitter returns a splitter that returns the next field.
func (s *Script) makeFieldSplitter() func([]byte, bool) (int, []byte, error) {
	// If the separator is empty, each rune is a separate field.
	if s.fs == "" {
		return bufio.ScanRunes
	}

	// If the separator is a single space, return the next word as the
	// field.
	if s.fs == " " {
		return bufio.ScanWords
	}

	// If the separator is a single character and the record terminator is
	// not empty (a special case in AWK), split based on that.  This code
	// is derived from the bufio.ScanWords source.
	if utf8.RuneCountInString(s.fs) == 1 && s.rs != "" {
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

	// If the separator is multiple characters (or the record terminator is
	// empty), treat it as a regular expression, and scan based on that.
	// First, we ensure the separator character is valid.
	var sepRegexp *regexp.Regexp
	var err error
	if s.rs == "" {
		// A special case in AWK is that if the record terminator is
		// empty (implying a blank line) then newlines are accepted as
		// a field separator in addition to whatever is specified for
		// FS.
		sepRegexp, err = s.compileRegexp(`(` + s.fs + `)|(\r?\n)`)
	} else {
		sepRegexp, err = s.compileRegexp(s.fs)
	}
	if err != nil {
		return func(data []byte, atEOF bool) (int, []byte, error) {
			return 0, nil, err
		}
	}

	// The regular expression is valid.  Return a splitter customized to
	// that regular expression.
	returnedFinalToken := false // true=already returned a final, non-terminated token; false=didn't
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// If we match the regular expression, return everything up to
		// the match.
		loc := sepRegexp.FindIndex(data)
		if loc != nil {
			return loc[1], data[:loc[0]], nil
		}

		// We didn't see a separator.  If we're at EOF, we have a
		// final, non-terminated token.  Return it (unless we already
		// did).
		if atEOF && !returnedFinalToken {
			returnedFinalToken = true
			return len(data), data, nil
		}

		// Request more data.
		return 0, nil, nil
	}
}

// makeRecordSplitter returns a splitter that returns the next record.
// Although all the AWK documentation I've read define RS as a record
// separator, as far as I can tell, AWK in fact treats it as a record
// *terminator* so we do, too.
func (s *Script) makeRecordSplitter() func([]byte, bool) (int, []byte, error) {
	// If the terminator is a single character, scan based on that.  This
	// code is derived from the bufio.ScanWords source.
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
			s.RT = string(firstRune)
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

	// If the terminator is multiple characters, treat it as a regular
	// expression, and scan based on that.  Or, as a special case, if the
	// terminator is empty, we treat it as a regular expression
	// representing one or more blank lines.
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// Generate a regular expression based on the current RS and
		// IgnoreCase.
		var termRegexp *regexp.Regexp
		if s.rs == "" {
			termRegexp, err = s.compileRegexp(`\r?\n(\r?\n)+`)
		} else {
			termRegexp, err = s.compileRegexp(s.rs)
		}
		if err != nil {
			return 0, nil, err
		}

		// If we match the regular expression, return everything up to
		// the match.
		loc := termRegexp.FindIndex(data)
		if loc != nil {
			s.RT = string(data[loc[0]:loc[1]])
			return loc[1], data[:loc[0]], nil
		}

		// We didn't see a terminator.  If we're at EOF, we have a
		// final, non-terminated token.  Return it if it's nonempty.
		if atEOF && len(data) > 0 {
			s.RT = ""
			return len(data), data, nil
		}

		// Request more data.
		return 0, nil, nil
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

// Split a record into fields.  Store the fields in the Script struct's F field
// and update NF.  As in real AWK, field 0 is the entire record.
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
	s.NF = len(fields) - 1
	return nil
}

// Execute a script against a given input stream.  It is perfectly valid to run
// the same script on multiple input streams.
func (s *Script) Run(r io.Reader) error {
	// Define a helper function that makes a pass through all user-defined
	// statements.
	walkStatements := func() {
		for _, rule := range s.rules {
			if rule.Pattern(s) {
				rule.Action(s)
				if s.stop != dontStop {
					return
				}
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

	// Process all Begin actions.
	s.state = atBegin
	walkStatements()
	if s.stop == stopScript {
		return nil
	}

	// Create (and store) a new scanner based on the record terminator.
	s.rsScanner = bufio.NewScanner(s.input)
	s.rsScanner.Split(s.makeRecordSplitter())

	// Process each record in turn.
	s.state = inMiddle
	for {
		// Read a record.
		s.stop = dontStop
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
		if s.stop == stopScript {
			return nil
		}
	}

	// Process all End actions
	s.state = atEnd
	walkStatements()
	s.state = notRunning
	return nil
}
