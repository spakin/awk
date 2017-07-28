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

// A scriptAborter is an error that causes the current script to abort but lets
// the rest of the program run.
type scriptAborter struct{ error }

// A recordStopper is thrown when a script wants to continue immediately with
// the next record.
type recordStopper struct{ error }

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

// Choose arbitrary initial sizes for record and field buffers.
const (
	initialFieldSize  = 4096
	initialRecordSize = 4096
)

// A Script encapsulates all of the internal state for an AWK-like script.
type Script struct {
	State         interface{} // Arbitrary, user-supplied data
	Output        io.Writer   // Output stream (defaults to os.Stdout)
	Begin         ActionFunc  // Action to perform before any input is read
	End           ActionFunc  // Action to perform after all input is read
	ConvFmt       string      // Conversion format for numbers, "%.6g" by default
	SubSep        string      // Separator for simulated multidimensional arrays
	NR            int         // Number of input records seen so far
	NF            int         // Number of fields in the current input record
	RT            string      // Actual string terminating the current record
	RStart        int         // 1-based index of the previous regexp match (Value.Match)
	RLength       int         // Length of the previous regexp match (Value.Match)
	MaxRecordSize int         // Maximum number of characters allowed in each record
	MaxFieldSize  int         // Maximum number of characters allowed in each field

	nf0          int                       // Value of NF for which F(0) was computed
	rs           string                    // Input record separator, newline by default
	fs           string                    // Input field separator, space by default
	fieldWidths  []int                     // Fixed-width column sizes
	fPat         string                    // Input field regular expression
	ors          string                    // Output record separator, newline by default
	ofs          string                    // Output field separator, space by default
	ignCase      bool                      // true: REs are case-insensitive; false: case-sensitive
	rules        []statement               // List of pattern-action pairs to execute
	fields       []*Value                  // Fields in the current record; fields[0] is the entire record
	regexps      map[string]*regexp.Regexp // Map from a regular-expression string to a compiled regular expression
	getlineState map[io.Reader]*Script     // Parsing state needed to invoke GetLine repeatedly on a given io.Reader
	rsScanner    *bufio.Scanner            // Scanner associated with RS
	input        io.Reader                 // Script input stream
	state        parseState                // What we're currently parsing
	stop         stopState                 // What we should stop doing
}

// NewScript initializes a new Script with default values.
func NewScript() *Script {
	return &Script{
		Output:        os.Stdout,
		ConvFmt:       "%.6g",
		SubSep:        "\034",
		NR:            0,
		NF:            0,
		MaxRecordSize: bufio.MaxScanTokenSize,
		MaxFieldSize:  bufio.MaxScanTokenSize,
		nf0:           0,
		rs:            "\n",
		fs:            " ",
		ors:           "\n",
		ofs:           " ",
		ignCase:       false,
		rules:         make([]statement, 0, 10),
		fields:        make([]*Value, 0),
		regexps:       make(map[string]*regexp.Regexp, 10),
		getlineState:  make(map[io.Reader]*Script),
		state:         notRunning,
	}
}

// abortScript aborts the current script with a formatted error message.
func (s *Script) abortScript(format string, a ...interface{}) {
	s.stop = stopScript
	panic(scriptAborter{fmt.Errorf(format, a...)})
}

// Return a copy of a Script.
func (s *Script) Copy() *Script {
	sc := *s
	sc.rules = make([]statement, len(s.rules))
	copy(sc.rules, s.rules)
	sc.fieldWidths = make([]int, len(s.fieldWidths))
	copy(sc.fieldWidths, s.fieldWidths)
	sc.fields = make([]*Value, len(s.fields))
	copy(sc.fields, s.fields)
	sc.regexps = make(map[string]*regexp.Regexp, len(s.regexps))
	for k, v := range s.regexps {
		sc.regexps[k] = v
	}
	sc.getlineState = make(map[io.Reader]*Script, len(s.getlineState))
	for k, v := range s.getlineState {
		sc.getlineState[k] = v
	}
	return &sc
}

// SetRS sets the input record separator (really, a record terminator).  It is
// invalid to call SetRS after the first record is read.  (It is acceptable to
// call SetRS from a Begin action, though.)  As in AWK, if the record separator
// is a single character, that character is used to separate records; if the
// record separator is multiple characters, it's treated as a regular
// expression (subject to the current setting of Script.IgnoreCase); and if the
// record separator is an empty string, records are separated by blank lines.
// That last case implicitly causes newlines to be accepted as a field
// separator in addition to whatever was specified by SetFS.
func (s *Script) SetRS(rs string) {
	if s.state == inMiddle {
		s.abortScript("SetRS was called from a running script")
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
func (s *Script) SetFS(fs string) {
	s.fs = fs
	s.fieldWidths = nil
	s.fPat = ""
}

// SetFieldWidths indicates that each record is composed of fixed-width columns
// and specifies the width in characters of each column.  It is invalid to pass
// SetFieldWidths a nil argument or a non-positive field width.
func (s *Script) SetFieldWidths(fw []int) {
	// Sanity-check the argument.
	if fw == nil {
		s.abortScript("SetFieldWidths was passed a nil slice")
	}
	for _, w := range fw {
		if w <= 0 {
			s.abortScript(fmt.Sprintf("SetFieldWidths was passed an invalid field width (%d)", w))
		}
	}

	// Assign the field widths and reset the field separator and field
	// matcher (not strictly but consistent with the SetFS method).
	s.fs = " "
	s.fieldWidths = fw
	s.fPat = ""
}

// SetFPat defines a "field pattern", a regular expression that matches fields.
// This lies in contrast to providing a regular expression to SetFS, which
// matches the separation between fields, not the fields themselves.
func (s *Script) SetFPat(fp string) {
	s.fs = " "
	s.fieldWidths = nil
	s.fPat = fp
}

// recomputeF0 recomputes F(0) by concatenating F(1)...F(NF) with OFS.
func (s *Script) recomputeF0() {
	if len(s.fields) >= 1 {
		s.fields[0] = s.NewValue(strings.Join(s.FStrings(), s.ofs))
	}
	s.nf0 = s.NF
}

// SetORS sets the output record separator.
func (s *Script) SetORS(ors string) { s.ors = ors }

// SetOFS sets the output field separator.
func (s *Script) SetOFS(ofs string) {
	s.ofs = ofs
	s.recomputeF0()
}

// F returns a specified field of the current record.  Field numbers are
// 1-based.  Field 0 refers to the entire record.  Requesting a field greater
// than NF returns a zero value.  Requesting a negative field number panics
// with an out-of-bounds error.
func (s *Script) F(i int) *Value {
	if i == 0 && s.NF != s.nf0 {
		s.recomputeF0()
	}
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

	// Force F(0) to be recomputed the next time it's accessed.
	s.nf0 = -1
}

// FStrings returns all fields in the current record as a []string of length
// NF.
func (s *Script) FStrings() []string {
	a := make([]string, s.NF)
	for i := 0; i < s.NF; i++ {
		a[i] = s.F(i + 1).String()
	}
	return a
}

// FInts returns all fields in the current record as a []int of length NF.
func (s *Script) FInts() []int {
	a := make([]int, s.NF)
	for i := 0; i < s.NF; i++ {
		a[i] = s.F(i + 1).Int()
	}
	return a
}

// FFloat64s returns all fields in the current record as a []float64 of length
// NF.
func (s *Script) FFloat64s() []float64 {
	a := make([]float64, s.NF)
	for i := 0; i < s.NF; i++ {
		a[i] = s.F(i + 1).Float64()
	}
	return a
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

// The matchAny pattern is true only in the middle of a script, when a record
// is available for parsing.
func matchAny(s *Script) bool {
	return s.state == inMiddle
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
	panic(recordStopper{errors.New("Unexpected Next invocation")}) // Unexpected if we don't catch it
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

// Auto provides a simplified mechanism for creating various common-case
// PatternFunc functions.  It accepts zero, one, or an even number of
// arguments.  If given no arguments, it matches every record.  If given a
// single argument, its behavior depends on that argument's type:
//
// • A Script.PatternFunc is returned as is.
//
// • A *regexp.Regexp returns a function that matches that regular expression
// against the entire record.
//
// • A string is treated as a regular expression and behaves likewise.
//
// • An int returns a function that matches that int against NR.
//
// • Any other type causes a run-time panic.
//
// If given an even number of arguments, pairs of arguments are treated as
// ranges (cf. the Range function).  The PatternFunc returns true if the record
// lies within any of the ranges.
func Auto(v ...interface{}) PatternFunc {
	if len(v) == 0 {
		// No arguments: Match anything.
		return matchAny
	}
	if len(v)%2 == 0 {
		// Even number of arguments other than 0: Return a disjunction
		// of ranges.
		fList := make([]PatternFunc, len(v)/2)
		for i := 0; i < len(v); i += 2 {
			f1 := Auto(v[i])
			f2 := Auto(v[i+1])
			fList[i/2] = Range(f1, f2)
		}
		return func(s *Script) bool {
			// Return true iff any range is true.  Note that we
			// always evaluate every range to avoid confusing
			// results because of statefulness.
			m := false
			for _, f := range fList {
				if f(s) {
					m = true
				}
			}
			return m
		}
	}
	if len(v)%2 == 1 {
		// Single argument: Decide what to do based on its type.
		switch x := v[0].(type) {
		case PatternFunc:
			// Already a PatternFunc: Return it unmodified.
			return x
		case string:
			// String: Treat as a regular expression that matches
			// against F[0].
			return func(s *Script) bool {
				r, err := s.compileRegexp(x)
				if err != nil {
					s.abortScript(err.Error())
				}
				return r.MatchString(s.F(0).String())
			}
		case int:
			// Integer: Match against NR.
			return func(s *Script) bool {
				return s.NR == x
			}
		case *regexp.Regexp:
			// Regular expression: Convert to a string then,
			// dynamically, back to a regular expression.  This
			// enables dynamic toggling of case sensitivity.
			xs := x.String()
			return func(s *Script) bool {
				r, err := s.compileRegexp(xs)
				if err != nil {
					s.abortScript(err.Error())
				}
				return r.MatchString(s.F(0).String())
			}
		default:
			panic(fmt.Sprintf("Auto does not accept arguments of type %T", x))
		}
	}
	panic("Auto expects 0, 1, or an even number of arguments")
}

// AppendStmt appends a pattern-action pair to a Script.  If the pattern
// function is nil, the action will be performed on every record.  If the
// action function is nil, the record will be output verbatim to the standard
// output device.  It is invalid to call AppendStmt from a running script.
func (s *Script) AppendStmt(p PatternFunc, a ActionFunc) {
	// Panic if we were called on a running script.
	if s.state != notRunning {
		s.abortScript("AppendStmt was called from a running script")
	}

	// Append a statement to the list of rules.
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

// makeSingleCharFieldSplitter returns a splitter that returns the next field
// by splitting on a single character (except for space, which is a special
// case handled elsewhere).
func (s *Script) makeSingleCharFieldSplitter() func([]byte, bool) (int, []byte, error) {
	// Ensure the separator character is valid.
	firstRune, _ := utf8.DecodeRuneInString(s.fs)
	if firstRune == utf8.RuneError {
		return func(data []byte, atEOF bool) (int, []byte, error) {
			return 0, nil, errors.New("Invalid rune in separator")
		}
	}

	// The separator is valid.  Return a splitter customized to that
	// separator.
	returnedFinalToken := false // true=already returned a final, non-terminated token; false=didn't
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// Scan until we see a separator or run out of data.
		for width, i := 0, 0; i < len(data); i += width {
			var r rune
			r, width = utf8.DecodeRune(data[i:])
			if r == utf8.RuneError && i+width >= len(data) {
				// Invalid rune at the end of the data.
				// Request more data and try again.
				return 0, nil, nil
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

// makeREFieldSplitter returns a splitter that returns the next field by
// splitting on a regular expression.
func (s *Script) makeREFieldSplitter() func([]byte, bool) (int, []byte, error) {
	// Ensure that the regular expression is valid.
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

// makeFixedFieldSplitter returns a splitter than returns the next field by
// splitting a record into fixed-size chunks.
func (s *Script) makeFixedFieldSplitter() func([]byte, bool) (int, []byte, error) {
	f := 0                      // Index into s.fieldWidths
	returnedFinalToken := false // true=already returned a final, non-terminated token; false=didn't
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// If we've exhausted s.fieldWidths, return empty-handed.
		if f >= len(s.fieldWidths) {
			return 0, nil, nil
		}

		// If we have enough characters for the current field, return a
		// token and advance to the next field.
		fw := s.fieldWidths[f]
		if len(data) >= fw {
			f++
			return fw, data[:fw], nil
		}

		// If we don't have enough characters for the current field but
		// we're at EOF, return whatever we have (unless we already
		// did).
		if atEOF && !returnedFinalToken {
			returnedFinalToken = true
			return len(data), data, nil
		}

		// If we don't have enough characters for the current field and
		// we're not at EOF, request more data.
		return 0, nil, nil
	}
}

// makeREFieldMatcher returns a splitter that returns the next field by
// matching against a regular expression.
func (s *Script) makeREFieldMatcher() func([]byte, bool) (int, []byte, error) {
	// Ensure that the regular expression is valid.
	sepRegexp, err := s.compileRegexp(s.fPat)
	if err != nil {
		return func(data []byte, atEOF bool) (int, []byte, error) {
			return 0, nil, err
		}
	}

	// The regular expression is valid.  Return a splitter customized to
	// that regular expression.
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// If we match the regular expression, return the match.
		// Otherwise, request more data.
		loc := sepRegexp.FindIndex(data)
		if loc == nil {
			return 0, nil, nil
		}
		return loc[1], data[loc[0]:loc[1]], nil
	}
}

// makeFieldSplitter returns a splitter that returns the next field.
func (s *Script) makeFieldSplitter() func([]byte, bool) (int, []byte, error) {
	// If we were given fixed field widths, use them.
	if s.fieldWidths != nil {
		return s.makeFixedFieldSplitter()
	}

	// If were given a field-matching regular expression, use it.
	if s.fPat != "" {
		return s.makeREFieldMatcher()
	}

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
		return s.makeSingleCharFieldSplitter()
	}

	// If the separator is multiple characters (or the record terminator is
	// empty), treat it as a regular expression, and scan based on that.
	return s.makeREFieldSplitter()
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
				if r == utf8.RuneError && i+width >= len(data) {
					// Invalid rune at the end of the data.
					// Request more data and try again.
					return 0, nil, nil
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

// splitRecord splits a record into fields.  It stores the fields in the Script
// struct's F field and update NF.  As in real AWK, field 0 is the entire
// record.
func (s *Script) splitRecord(rec string) error {
	fsScanner := bufio.NewScanner(strings.NewReader(rec))
	fsScanner.Buffer(make([]byte, initialFieldSize), s.MaxFieldSize)
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
	s.nf0 = s.NF
	return nil
}

// GetLine reads the next record from an input stream and returns it.  If the
// argument to GetLine is nil, GetLine reads from the current input stream and
// increments NR.  Otherwise, it reads from the given io.Reader and does not
// increment NR.  Call SetF(0, ...) on the Value returned by GetLine to perform
// the equivalent of AWK's getline with no variable argument.
func (s *Script) GetLine(r io.Reader) (*Value, error) {
	// Handle the simpler case of a nil argument (to read from the current
	// input stream).
	if r == nil {
		rec, err := s.readRecord()
		if err != nil {
			return nil, err
		}
		s.NR++
		return s.NewValue(rec), nil
	}

	// If we've seen this io.Reader before, reuse its parsing state.
	// Otherwise, create a new Script for storing state.
	sc := s.getlineState[r]
	if sc == nil {
		// Copy the given script so we don't alter any of the original
		// script's state.
		sc = s.Copy()
		s.getlineState[r] = sc

		// Create (and store) a new scanner based on the record
		// terminator.
		sc.input = r
		sc.rsScanner = bufio.NewScanner(sc.input)
		sc.rsScanner.Buffer(make([]byte, initialRecordSize), sc.MaxRecordSize)
		sc.rsScanner.Split(sc.makeRecordSplitter())
	}

	// Read a record from the given reader.
	rec, err := sc.readRecord()
	if err != nil {
		return nil, err
	}
	return sc.NewValue(rec), nil
}

// Run executes a script against a given input stream.  It is perfectly valid
// to run the same script on multiple input streams.
func (s *Script) Run(r io.Reader) (err error) {
	// Catch scriptAborter panics and return them as errors.  Re-throw all
	// other panics.
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(scriptAborter); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()

	// Reinitialize most of our state.
	s.input = r
	s.ConvFmt = "%.6g"
	s.NF = 0
	s.NR = 0

	// Process the Begin action, if any.
	if s.Begin != nil {
		s.state = atBegin
		s.Begin(s)
	}

	// Create (and store) a new scanner based on the record terminator.
	s.rsScanner = bufio.NewScanner(s.input)
	s.rsScanner.Buffer(make([]byte, initialRecordSize), s.MaxRecordSize)
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
		err = s.splitRecord(rec)
		if err != nil {
			return err
		}

		// Process all applicable actions.
		func() {
			// An action is able to break out of the
			// action-processing loop by calling Next, which throws
			// a recordStopper.  We catch that and continue
			// with the next record.
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(recordStopper); !ok {
						panic(r)
					}
				}
			}()

			// Perform each action whose pattern matches the
			// current record.
			for _, rule := range s.rules {
				if rule.Pattern(s) {
					rule.Action(s)
					if s.stop != dontStop {
						break
					}
				}
			}
		}()

		// Stop the script if an error occurred or an action calls  Exit.
		if s.stop == stopScript {
			return nil
		}
	}

	// Process the End action, if any.
	if s.End != nil {
		s.state = atEnd
		s.End(s)
	}
	s.state = notRunning
	return nil
}
