awk
===

Description
-----------

`awk` is a package for the [Go programming language](https://golang.org/) that provides an [AWK](http://pubs.opengroup.org/onlinepubs/9699919799/utilities/awk.html)-style text processing capability.  The package facilitates splitting an input stream into records (default: newline-separated lines) and fields (default: whitespace-separated columns) then applying a sequence of statements of the form "if 〈_pattern_〉 then 〈_action_〉" to each record in turn.  For example, the following is a complete Go program that adds up the first two columns of a [CSV](https://en.wikipedia.org/wiki/Comma-separated_values) file to produce a third column:
```Go
package main

import (
    "github.com/spakin/awk"
    "os"
)

func main() {
    s := awk.NewScript()
    s.AppendStmt(awk.Begin, func(s *awk.Script) {
        s.SetFS(",")
        s.SetOFS(",")
    })
    s.AppendStmt(nil, func(s *awk.Script) {
        s.SetF(3, s.NewValue(s.F(1).Int()+s.F(2).Int()))
        s.Println()
    })
    s.Run(os.Stdin)
}
```

In the above, the `awk` package handles all the mundane details such as reading lines from the file, checking for EOF, splitting lines into columns, handling errors, and other such things.  With the help of `awk`, Go easily can be applied to the sorts of text-processing tasks that one would normally implement in a scripting language but without sacrificing Go's speed, safety, or flexibility.

Installation
------------

Instead of manually downloading and installing `awk` from GitHub, the recommended approach is to ensure your `GOPATH` environment variable is set properly then issue a
```bash
go get github.com/spakin/awk
```
command.

Author
------

[Scott Pakin](http://www.pakin.org/~scott/), *scott+awk@pakin.org*
