package generator

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

type BlacklistWriter interface {
	Write(h []string) error
}

func Parse(blacklistWriter BlacklistWriter, source ...string) error {
	hosts := make(map[string]struct{})

	// Use stdin if there are no sources specified or if the source is a single dash.
	if (len(source) == 1 && source[0] == "-") || len(source) == 0 {
		if err := parse(os.Stdin, hosts); err != nil {
			return err
		}
	} else {
		// Loop through each source and check if it's an URI and fall back to normal file operation if not.
		for _, filename := range source {
			loc, err := url.ParseRequestURI(filename)
			if err == nil && loc.IsAbs() {
				if err := parseRemoteSource(*loc, hosts, parse); err != nil {
					return err
				}
			} else if err != nil {
				if err := parseLocalSource(filename, hosts, parse); err != nil {
					return err
				}
			}
		}
	}

	names := make([]string, 0, len(hosts))
	for k := range hosts {
		names = append(names, k)
	}
	sort.Strings(names)

	return blacklistWriter.Write(names)
}

func parseLocalSource(filename string, hosts map[string]struct{}, fn func(r io.Reader, hosts map[string]struct{}) error) error {
	file, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}(file)

	return fn(file, hosts)
}

func parseRemoteSource(loc url.URL, hosts map[string]struct{}, fn func(r io.Reader, hosts map[string]struct{}) error) error {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	if strings.EqualFold(loc.Scheme, "file") {
		t := &http.Transport{}
		t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
		client.Transport = t
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, loc.String(), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer client.CloseIdleConnections()
	defer func(r io.ReadCloser) {
		if err := resp.Body.Close(); err != nil {
			panic(err)
		}
	}(resp.Body)

	return fn(resp.Body, hosts)
}

func parse(r io.Reader, hosts map[string]struct{}) error {
	s := bufio.NewScanner(r)

	for s.Scan() {
		if s.Err() != nil {
			return s.Err()
		}

		// Trim leading whitespaces.
		line := strings.TrimLeftFunc(s.Text(), unicode.IsSpace)

		// Skip lines that begin with a #.
		if strings.HasPrefix(line, "#") || len(line) == 0 {
			continue
		}

		// Cut line at the first #.
		if trim := strings.IndexRune(line, '#'); trim > 0 {
			line = line[:trim]
		}

		// Split into fields.
		fields := strings.FieldsFunc(line, unicode.IsSpace)

		// If length == 1 then it's safe to assume that it's just a list of host names
		// but if length == 2 then it's safe to assume that it's a hosts file.
		if len(fields) == 1 {
			if net.ParseIP(fields[0]) == nil {
				hosts[strings.ToLower(fields[0])] = struct{}{}
			}
		} else if len(fields) == 2 {
			if addr := net.ParseIP(fields[0]); addr == nil {
				if fields[1] == "localhost" || strings.HasPrefix(fields[1], "localhost.") {
					continue
				}
				return fmt.Errorf("unrecognized format of line: %q", line)
			} else if addr.IsUnspecified() ||
				(addr.IsLoopback() && !(strings.HasPrefix(fields[1], "localhost") || strings.HasPrefix(fields[1], "localhost."))) {
				if net.ParseIP(fields[1]) == nil {
					hosts[strings.ToLower(fields[1])] = struct{}{}
				}
			}
		} else {
			return fmt.Errorf("unrecognized format of line: %q", line)
		}
	}
	return nil
}
