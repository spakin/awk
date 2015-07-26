// This file presents some examples of awk package usage.

package awk_test

import (
	"fmt"
	"github.com/spakin/awk"
)

var s *awk.Script

// Write to the standard output all input lines for which field 3 is
// greater than 5 (AWK: $3 > 5).
func Example_01() {
	s.AppendStmt(func(s *awk.Script) bool { return s.F(3).Int() > 5 }, nil)
}

// Write every tenth line (AWK: (NR % 10) == 0).
func Example_02() {
	s.AppendStmt(func(s *awk.Script) bool { return s.NR%10 == 0 }, nil)
}

// Write any line with a substring containing a 'G' or 'D', followed by a
// sequence of digits and characters (AWK:
// /(G|D)([[:digit:][:alpha:]]*)/). This example uses character classes digit
// and alpha to match language-independent digit and alphabetic characters
// respectively.
func Example_04() {
	s.AppendStmt(func(s *awk.Script) bool { return s.F(0).Match("(G|D)([[:digit:][:alpha:]]*)") }, nil)
}

// Write any line in which the second field matches the regular expression
// "xyz" and the fourth field does not (AWK: $2 ~ /xyz/ && $4 !~ /xyz/).
func Example_05() {
	s.AppendStmt(func(s *awk.Script) bool {
		return s.F(2).Match("xyz") && !s.F(4).Match("xyz")
	}, nil)
}

// Write any line in which the second field contains a backslash (AWK: $2
// /\\/).
func Example_06() {
	s.AppendStmt(func(s *awk.Script) bool { return s.F(2).Match(`\\`) }, nil)
}

// Write the second to the last and the last field in each line. Separate the
// fields by a colon (AWK: {OFS=":"; print $(NF-1), $NF}).
func Example_08() {
	s.AppendStmt(nil, func(s *awk.Script) { fmt.Printf("%v:%v\n", s.F(s.NF-1), s.F(s.NF)) })
}

// Write the line number and number of fields in each line (AWK: {print NR ":"
// NF}). The three strings representing the line number, the colon, and the
// number of fields are concatenated and that string is written to standard
// output.
func Example_09() {
	s.AppendStmt(nil, func(s *awk.Script) { fmt.Printf("%d:%d\n", s.NR, s.NF) })
}

// Write lines longer than 72 characters (AWK: length($0) > 72).
func Example_10() {
	s.AppendStmt(func(s *awk.Script) bool { return len(s.F(0).String()) > 72 }, nil)
}

// Write the first two fields in opposite order (AWK: {print $2, $1}).
func Example_11() {
	s.AppendStmt(nil, func(s *awk.Script) { fmt.Printf("%v %v\n", s.F(2), s.F(1)) })
}

// Do the same as Example 11, with input fields separated by a comma, space and
// tab characters, or both (AWK:
//
//     BEGIN { FS = ",[ \t]*|[ \t]+" }
//           { print $2, $1 }
//
// ).
func Example_12() {
	s.AppendStmt(awk.Begin, func(s *awk.Script) { s.SetFS(",[ \t]*|[ \t]+") })
	s.AppendStmt(nil, func(s *awk.Script) { fmt.Printf("%v %v\n", s.F(2), s.F(1)) })
}

// Add up the first column and print the sum and average (AWK:
//
//     {s += $1 }
// END {print "sum is", s, "average is", s/NR}
//
// ).
func Example_13() {
	sum := 0.0
	s.AppendStmt(nil, func(s *awk.Script) { sum += s.F(1).Float64() })
	s.AppendStmt(awk.End, func(s *awk.Script) {
		fmt.Println("sum is", sum, "average is", sum/float64(s.NR))
	})
}

// Write fields in reverse order, one per line (many lines out for each line
// in).  AWK: {for (i = NF; i > 0; --i) print $i}.
func Example_14() {
	s.AppendStmt(nil, func(s *awk.Script) {
		for i := s.NF; i > 0; i-- {
			fmt.Println(s.F(i))
		}
	})
}

// Write all lines between occurrences of the strings start and stop (AWK:
// /start/, /stop/).
func Example_15() {
	s.AppendStmt(awk.Range(func(s *awk.Script) bool { return s.F(1).Match("start") },
		func(s *awk.Script) bool { return s.F(1).Match("stop") }),
		nil)
}
