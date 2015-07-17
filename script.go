package awk

// A Script contains all the internal state for an AWK-like script.
type Script struct {
	ConvFmt string // Conversion format for numbers, "%.6g" by default
	FS      string // Input field separator, space by default
	NF      int    // Number of fields in the current input record
	NR      int    // Number of input records seen so far
	RS      string // Input record separator, newline by default
}

// NewScript initializes a new Script with default values.
func NewScript() *Script {
	return &Script{
		ConvFmt: "%.6g",
		FS:      " ",
		NF:      0,
		NR:      0,
		RS:      "\n",
	}
}
