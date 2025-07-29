package app

import (
	"bufio"
	"fmt"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"os"
	"strings"
)

// ReadMapping reads a mapping file and returns a map of hostnames to backend
// addresses.
//
// # Parameters
//
// - file (string): The path to the mapping file to read.
//
// # Return Values
//
// - m (map[string]string): A map containing the hostname to backend address
// mappings parsed from the file.
//
// - err (error): An error if any step during reading or parsing fails.
//
// # Expected behaviour
//
// - Opens the specified file and reads its contents line by line.
//
// - Skips lines that are empty or start with a '#'.
//
// - Splits each valid line into two parts using the first colon as the
// separator.
//
// - Trims whitespace from both parts and adds them to the map.
//
// - Returns any error encountered during file operations or parsing.
func ReadMapping(file string) (m map[string]string, err error) {
	var f *os.File
	if f, err = os.Open(file); chk.E(err) {
		return
	}
	m = make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if b := sc.Bytes(); len(b) == 0 || b[0] == '#' {
			continue
		}
		s := strings.SplitN(sc.Text(), ":", 2)
		if len(s) != 2 {
			err = fmt.Errorf("invalid line: %q", sc.Text())
			log.E.Ln(err)
			chk.E(f.Close())
			return
		}
		m[strings.TrimSpace(s[0])] = strings.TrimSpace(s[1])
	}
	err = sc.Err()
	chk.E(err)
	chk.E(f.Close())
	return
}
