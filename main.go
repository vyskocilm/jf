package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

const (
	exitNoDiff   = 0
	exitDiff     = 1
	exitTroubles = 2
)

func js(pathA, pathB string) (string, string, error) {
	bytesA, err := ioutil.ReadFile(pathA)
	if err != nil {
		return "", "", err
	}
	bytesB, err := ioutil.ReadFile(pathB)
	if err != nil {
		return "", "", err
	}

	jsA := string(bytesA)
	jsB := string(bytesB)

	return jsA, jsB, nil
}

func main() {

	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: jf a.json b.json\n")
		os.Exit(exitTroubles)
	}
	jsA, jsB, err := js(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
        os.Exit(exitTroubles)
	}

	lines, err := diff(jsA, jsB)
	for _, p := range lines {
		fmt.Printf("%s\t%s\t%s\n", p.selector, p.valueA, p.valueB)
	}

    if len(lines) == 0 {
        os.Exit(exitNoDiff)
    }
    os.Exit(exitDiff)
}
