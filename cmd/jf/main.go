// jf: quick and dirty CLI for jf (jsondiff) library
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"text/tabwriter"

	"github.com/vyskocilm/jf"
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

func makeRules(d *jf.Differ, ignoreB, sortAllBySelector, sortAllByKey *string) error {

	if *ignoreB != "" {
		rg, err := regexp.Compile(*ignoreB)
		if err != nil {
			return err
		}
		d.AddIgnore(jf.RuleB, rg)
	}

	if *sortAllBySelector != "" && *sortAllByKey != "" {
		rg, err := regexp.Compile(*sortAllBySelector)
		if err != nil {
			return err
		}
		d.AddOrderByKey(jf.RuleAB, rg, *sortAllByKey)
	}

	return nil
}

func main() {

	// FIXME: specify meaningful cmd arguments
	var (
		ignoreB           = flag.String("x-ignore-b", "", "ignore keys from b.json")
		sortAllBySelector = flag.String("x-sort-all-selector", "", "selector for slices to be sorted")
		sortAllByKey      = flag.String("x-sort-all-key", "", "key for sorting")
	)
	flag.Parse()

	d := jf.NewDiffer()
	err := makeRules(d, ignoreB, sortAllBySelector, sortAllByKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing commandline flags: %s", err)
		os.Exit(exitTroubles)
	}

	if len(flag.Args()) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: jf a.json b.json\n")
		os.Exit(exitTroubles)
	}
	jsA, jsB, err := js(flag.Arg(0), flag.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(exitTroubles)
	}

	diff, err := d.Diff(jsA, jsB)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, p := range diff {
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.Selector(), p.A(), p.B())
	}

	if len(diff) == 0 {
		os.Exit(exitNoDiff)
	}
	w.Flush()
	os.Exit(exitDiff)
}
