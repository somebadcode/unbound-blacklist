package generator

import (
	"bufio"
	"fmt"
	"os"
	"path"
)

type BlacklistWriterUnbound struct {
	output string
}

func New(output string) *BlacklistWriterUnbound {
	return &BlacklistWriterUnbound{
		output: output,
	}
}

// Write blacklist for Unbound.
func (b BlacklistWriterUnbound) Write(hosts []string) error {
	var f *os.File
	var err error

	if len(b.output) == 0 {
		f = os.Stdout
	} else {
		// If bl.path is a directory, we'll have to append a filename.
		if fi, err := os.Stat(b.output); os.IsPermission(err) {
			return err
		} else if os.IsExist(err) && fi.IsDir() {
			b.output = path.Join(b.output, "blacklist.conf")
		}

		// Create/truncate the file.
		f, err = os.OpenFile(b.output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
		if err != nil {
			return err
		}

		// Close and panic if closing fails.
		defer func(f *os.File) {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}(f)
	}

	// Buffered writing and defer flushing.
	w := bufio.NewWriter(f)
	defer func(w *bufio.Writer) {
		if err := w.Flush(); err != nil {
			panic(err)
		}
	}(w)

	// Start writing the hosts to the file.
	for _, host := range hosts {
		_, err := fmt.Fprintf(w, "local-zone: %s. always_nxdomain\n", host)
		if err != nil {
			return err
		}
	}

	return nil
}
