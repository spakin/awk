// This file presents some examples of awk package usage.

package awk_test

import (
	"fmt"
	"github.com/spakin/awk"
	"os"
	"sort"
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
	s.Begin = func(s *awk.Script) { s.SetFS(",[ \t]*|[ \t]+") }
	s.AppendStmt(nil, func(s *awk.Script) { fmt.Printf("%v %v\n", s.F(2), s.F(1)) })
}

// Add up the first column and print the sum and average (AWK:
//
//         {s += $1 }
//     END {print "sum is", s, "average is", s/NR}
//
// ).
func Example_13() {
	sum := 0.0
	s.AppendStmt(nil, func(s *awk.Script) { sum += s.F(1).Float64() })
	s.End = func(s *awk.Script) {
		fmt.Println("sum is", sum, "average is", sum/float64(s.NR))
	}
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

// Write all lines whose first field is different from the previous line's
// first field (AWK: $1 != prev {print; prev = $1}).
func Example_16() {
	prev := s.NewValue("")
	s.AppendStmt(func(s *awk.Script) bool { return !s.F(1).StrEqual(prev) },
		func(s *awk.Script) {
			fmt.Println(s.F(0))
			prev = s.F(1)
		})
}

// For all rows of the form "Total: <number>", accumulate <number>.  Once all
// rows have been read, output the grand total.
func ExampleScript_AppendStmt() {
	grandTotal := 0.0
	s := awk.NewScript()
	s.AppendStmt(func(s *awk.Script) bool { return s.NF == 2 && s.F(1).StrEqual("Total:") },
		func(s *awk.Script) { grandTotal += s.F(2).Float64() })
	s.End = func(s *awk.Script) { fmt.Printf("The grand total is %.2f\n", grandTotal) }
	s.Run(os.Stdin)
}

// Output each line preceded by its line number.
func ExampleScript_AppendStmt_nilPattern() {
	s := awk.NewScript()
	s.AppendStmt(nil, func(s *awk.Script) { fmt.Printf("%4d %v\n", s.NR, s.F(0)) })
	s.Run(os.Stdin)
}

// Output only rows in which the first column contains a larger number than the
// second column.
func ExampleScript_AppendStmt_nilAction() {
	s := awk.NewScript()
	s.AppendStmt(func(s *awk.Script) bool { return s.F(1).Int() > s.F(2).Int() }, nil)
	s.Run(os.Stdin)
}

// Output all input lines that appear between "BEGIN" and "END" inclusive.
func ExampleRange() {
	s := awk.NewScript()
	s.AppendStmt(awk.Range(func(s *awk.Script) bool { return s.F(1).StrEqual("BEGIN") },
		func(s *awk.Script) bool { return s.F(1).StrEqual("END") }),
		nil)
	s.Run(os.Stdin)
}

// Extract the first column of the input into a slice of strings.
func ExampleBegin() {
	var data []string
	s := awk.NewScript()
	s.Begin = func(s *awk.Script) {
		s.SetFS(",")
		data = make([]string, 0)
	}
	s.AppendStmt(nil, func(s *awk.Script) { data = append(data, s.F(1).String()) })
	s.Run(os.Stdin)
}

// Output each line with its columns in reverse order.
func ExampleScript_F() {
	s := awk.NewScript()
	s.AppendStmt(nil, func(s *awk.Script) {
		for i := s.NF; i > 0; i-- {
			if i > 1 {
				fmt.Printf("%v ", s.F(i))
			} else {
				fmt.Printf("%v\n", s.F(i))
			}
		}
	})
	s.Run(os.Stdin)
}

// Allocate and populate a 2-D array.  The diagonal is made up of strings while
// the rest of the array consists of float64 values.
func ExampleValueArray_Set() {
	va := s.NewValueArray()
	diag := []string{"Dasher", "Dancer", "Prancer", "Vixen", "Comet", "Cupid", "Dunder", "Blixem"}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			if i == j {
				va.Set(i, j, diag[i])
			} else {
				va.Set(i, j, float64(i*8+j)/63.0)
			}
		}
	}
}

// Sort each line's columns, which are assumed to be floating-point numbers.
func ExampleScript_FFloat64s() {
	s := awk.NewScript()
	s.AppendStmt(nil, func(s *awk.Script) {
		nums := s.FFloat64s()
		sort.Float64s(nums)
		for _, n := range nums[:len(nums)-1] {
			fmt.Printf("%.5g ", n)
		}
		fmt.Printf("%.5g\n", nums[len(nums)-1])
	})
	s.Run(os.Stdin)
}

// Delete the fifth line of the input stream but output all other lines.
func ExampleAuto_int() {
	s := awk.NewScript()
	s.AppendStmt(awk.Auto(5), func(s *awk.Script) { s.Next() })
	s.AppendStmt(nil, nil)
	s.Run(os.Stdin)
}

// Output only those lines containing the string, "fnord".
func ExampleAuto_string() {
	s := awk.NewScript()
	s.AppendStmt(awk.Auto("fnord"), nil)
	s.Run(os.Stdin)
}
