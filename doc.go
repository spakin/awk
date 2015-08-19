/*

Package awk implements AWK-style processing of input streams.


Introduction

AWK is a pattern scanning and processing language defined as part of the POSIX
1003.1 standard
(http://pubs.opengroup.org/onlinepubs/9699919799/utilities/awk.html) and
therefore part of all standard Linux/Unix distributions.  Its forte is simple
transformations of data arranged in rows and columns.  For example, the
following is a complete AWK program that reads an entire file from the standard
input device, splits each file into space-separated columns, and outputs all
lines in which the fifth column is an odd number:

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
	    s.Run(os.Stdin)
    }

While not a one-liner like the original AWK program, the above is conceptually
close to it.  The AppendStmt method defines a script in terms of patterns and
actions exactly as in the AWK program.  The Run method then runs the script on
an input stream, which can be any io.Reader.


Features

The awk package can be considered a shallow EDSL (embedded domain-specific
language) for Go that facilitates text processing.  The package handles the
reading and parsing of the input file and provides a few AWK-like data types to
further simplify code.  See the awk(1) manual page on any Linux/Unix system
(available online from, e.g., http://linux.die.net/man/1/awk) or read the book,
"The AWK Programming Language" by Aho, Kernighan, and Weinberger for more
information about the AWK language.  The following AWK features are currently
supported by the awk package:

• the basic pattern/action structure of an AWK script, including BEGIN and END
rules

• control over record separation (RS), including support for regular
expressions and null strings (implying blank lines as separators)

• control over field separation (FS), including support for regular expressions
and null strings (implying single-character fields)

• support for fixed-width fields (FIELDWIDTHS)

• support for fields defined by a regular expression (FPAT)

• control over case-sensitive vs. case-insensitive comparisons (IGNORECASE)

• control over the number conversion format (CONVFMT)

• automatic enumeration of records (NR) and fields (NR)

• "weak typing" support

• multi-dimensional associative arrays

• support for premature termination of record processing (next) and script
processing (exit)

• maintenance of regular-expression status variables (RT, RSTART, and RLENGTH)


Usage

Basic usage of the awk package comprises three steps:

1. Script allocation (awk.NewScript)

2. Script definition (Script.AppendStmt)

3. Script execution (Script.Run)

In Step 2, Script.AppendStmt is called once for each pattern/action
pair that is to be appended to the script.  The same script can be
applied to multiple input streams by re-executing Step 3 (even
concurrently, if desired).  Actions to be executed on every run of
Step 3 can be supplied by assigning the script's Begin and End fields.
The Begin action is typically used to initialize script state by
calling methods such as SetRS and SetFS and assigning user-defined
data to the script's State field (what would be global variables in
AWK).  The End action is typically used to store or report final
results.


Examples

A number of examples ported from the POSIX 1003.1 standard document
(http://pubs.opengroup.org/onlinepubs/9699919799/utilities/awk.html) are
presented below.

*/
package awk
