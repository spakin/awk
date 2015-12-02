/*

Package awk implements AWK-style processing of input streams.


Introduction

The awk package can be considered a shallow EDSL (embedded domain-specific
language) for Go that facilitates text processing.  It aims to implement
the core semantics provided by
AWK, a pattern scanning and processing language defined as part of the POSIX
1003.1 standard
(http://pubs.opengroup.org/onlinepubs/9699919799/utilities/awk.html) and
therefore part of all standard Linux/Unix distributions.

AWK's forte is simple transformations of tabular data.  For example, the
following is a complete AWK program that reads an entire file from the standard
input device, splits each file into whitespace-separated columns, and outputs
all lines in which the fifth column is an odd number:

    $5 % 2 == 1

Here's a typical Go analogue of that one-line AWK program:

    package main

    import (
            "bufio"
            "fmt"
            "io"
            "os"
            "strconv"
            "strings"
    )

    func main() {
            input := bufio.NewReader(os.Stdin)
            for {
                    line, err := input.ReadString('\n')
                    if err != nil {
                            if err != io.EOF {
                                    panic(err)
                            }
                            break
                    }
                    scanner := bufio.NewScanner(strings.NewReader(line))
                    scanner.Split(bufio.ScanWords)
                    cols := make([]string, 0, 10)
                    for scanner.Scan() {
                            cols = append(cols, scanner.Text())
                    }
                    if err := scanner.Err(); err != nil {
                            panic(err)
                    }
                    if len(cols) < 5 {
                            continue
                    }
                    num, err := strconv.Atoi(cols[4])
                    if num%2 == 1 {
                            fmt.Print(line)
                    }
            }
    }

The goal of the awk package is to emulate AWK's simplicity while simultaneously
taking advantage of Go's speed, safety, and flexibility.  With the awk package,
the preceding code reduces to the following:

    package main

    import (
	    "github.com/spakin/awk"
	    "os"
    )

    func main() {
	    s := awk.NewScript()
	    s.AppendStmt(func(s *awk.Script) bool { return s.F(5).Int()%2 == 1 }, nil)
	    if err := s.Run(os.Stdin); err != nil {
		    panic(err)
	    }
    }

While not a one-liner like the original AWK program, the above is conceptually
close to it.  The AppendStmt method defines a script in terms of patterns and
actions exactly as in the AWK program.  The Run method then runs the script on
an input stream, which can be any io.Reader.


Usage

For those programmers unfamiliar with AWK, an AWK program consists of a
sequence of pattern/action pairs.  Each pattern that matches a given line
causes the corresponding action to be performed.  AWK programs tend to be terse
because AWK implicitly reads the input file, splits it into records (default:
newline-terminated lines), and splits each record into fields (default:
whitespace-separated columns), saving the programmer from having to express
such operations explicitly.  Furthermore, AWK provides a default pattern, which
matches every record, and a default action, which outputs a record unmodified.

The awk package attempts to mimic those semantics in Go.  Basic usage consists
of three steps:

1. Script allocation (awk.NewScript)

2. Script definition (Script.AppendStmt)

3. Script execution (Script.Run)

In Step 2, AppendStmt is called once for each pattern/action pair that is to be
appended to the script.  The same script can be applied to multiple input
streams by re-executing Step 3.  Actions to be executed on every run of Step 3
can be supplied by assigning the script's Begin and End fields.  The Begin
action is typically used to initialize script state by calling methods such as
SetRS and SetFS and assigning user-defined data to the script's State field
(what would be global variables in AWK).  The End action is typically used to
store or report final results.

To mimic AWK's dynamic type system. the awk package provides the Value and
ValueArray types.  Value represents a scalar that can be coerced without error
to a string, an int, or a float64.  ValueArray represents a—possibly
multidimensional—associative array of Values.

Both patterns and actions can access the current record's fields via the
script's F method, which takes a 1-based index and returns the corresponding
field as a Value.  An index of 0 returns the entire record as a Value.


Features

The following AWK features and GNU AWK extensions are currently supported by
the awk package:

• the basic pattern/action structure of an AWK script, including BEGIN and END
rules and range patterns

• control over record separation (RS), including regular expressions and null
strings (implying blank lines as separators)

• control over field separation (FS), including regular expressions and null
strings (implying single-character fields)

• fixed-width fields (FIELDWIDTHS)

• fields defined by a regular expression (FPAT)

• control over case-sensitive vs. case-insensitive comparisons (IGNORECASE)

• control over the number conversion format (CONVFMT)

• automatic enumeration of records (NR) and fields (NR)

• "weak typing"

• multidimensional associative arrays

• premature termination of record processing (next) and script processing (exit)

• explicit record reading (getline) from either the current stream or
a specified stream

• maintenance of regular-expression status variables (RT, RSTART, and RLENGTH)

For more information about AWK and its features, see the awk(1) manual page on
any Linux/Unix system (available online from, e.g.,
http://linux.die.net/man/1/awk) or read the book, "The AWK Programming
Language" by Aho, Kernighan, and Weinberger.


Examples

A number of examples ported from the POSIX 1003.1 standard document
(http://pubs.opengroup.org/onlinepubs/9699919799/utilities/awk.html) are
presented below.

*/
package awk
