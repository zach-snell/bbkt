package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON formats any Go struct as pretty JSON and prints to stdout.
func PrintJSON(data any) {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
