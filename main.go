package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/tabwriter"
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

func makeRules(ignoreB, sortAllBySelector, sortAllByKey *string) ([]*rule, []*rule) {
	rulesA := make([]*rule, 0)
	rulesB := make([]*rule, 0)

	if *ignoreB != "" {
		r, err := newIgnoreRule(*ignoreB)
		if err != nil {
			log.Fatal(err)
		}
		rulesB = append(rulesB, r)
	}

	if *sortAllBySelector != "" && *sortAllByKey != "" {
		r, err := newOrderbyKeyRule(*sortAllBySelector, *sortAllByKey)
		if err != nil {
			log.Fatal(err)
		}
		rulesA = append(rulesA, r)
		rulesB = append(rulesB, r)
	}

	return rulesA, rulesB
}

func main() {

	// FIXME: specify meaningful cmd arguments
	var (
		ignoreB           = flag.String("x-ignore-b", "", "ignore keys from b.json")
		sortAllBySelector = flag.String("x-sort-all-selector", "", "selector for slices to be sorted")
		sortAllByKey      = flag.String("x-sort-all-key", "", "key for sorting")
	)
	flag.Parse()
	rulesA, rulesB := makeRules(ignoreB, sortAllBySelector, sortAllByKey)

	if len(flag.Args()) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: jf a.json b.json\n")
		os.Exit(exitTroubles)
	}
	jsA, jsB, err := js(flag.Arg(0), flag.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(exitTroubles)
	}

	lines, err := diff2(jsA, jsB, rulesA, rulesB)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, p := range lines {
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.selector, p.valueA, p.valueB)
	}

	if len(lines) == 0 {
		os.Exit(exitNoDiff)
	}
	w.Flush()
	os.Exit(exitDiff)
}
