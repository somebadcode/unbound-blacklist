package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/somebadcode/unbound-blacklist/pkg/generator"
	"github.com/spf13/pflag"
)

func main() {
	input := pflag.StringSliceP("input", "i", []string{}, "Hosts file(s) to convert (default is standard input)")
	output := pflag.StringP("output", "o", "", "Output file (default is standard output)")
	flag.Parse()

	writer := generator.New(*output)
	err := generator.Parse(writer, *input...)
	if err != nil {
		_, err = fmt.Fprintf(os.Stderr, "Failed trying to generate blacklist file: %s\n", err)
		if err != nil {
			panic(err)
		}
		os.Exit(1)
	}
}
