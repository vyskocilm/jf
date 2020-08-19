// jsondiff: diff for JSON
package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/stretchr/objx"
)

/*
    rename: rename key to key'
    exclude: exclude key from comparsion
    aString: convert numbers to string for comparsion
    coercenull: make null == false, "", {} or [], 0
*/
type ruleAction int
const (
    rename ruleAction = iota
    exclude
    asString
    coercenull
)

type rule struct {
    selector string
    action ruleAction
    newKey string
    excludedKey string
}

func newCoerceNullRule(selector string) *rule {
    return &rule{selector: selector, action: coercenull}
}

func (r *rule) match(selector string) bool {
    if r.selector != "*" {
        panic("given selector is not implemented")
    }
    return true
}

// patch contains data about single difference
type patch struct {
	selector string
	valueA   string
	valueB   string
}

type patchLines []*patch

type differ struct {
	lines patchLines
    rulesA []*rule
    rulesB []*rule
}

type stringer struct {
	i interface{}
}

func (s stringer) String() string {
	return fmt.Sprintf("%+v", s.i)
}

func joinSelectors(mainSelector, selector string) string {
	if mainSelector == "" {
		return selector
	}
	if strings.HasPrefix(selector, "[") {
		return mainSelector + selector
	}
	return mainSelector + "." + selector
}

// lineA adds line with empty B value
func (d *differ) lineA(mainSelector, selector string, valueA fmt.Stringer) {
	d.lines = append(
		d.lines,
		&patch{
			selector: joinSelectors(mainSelector, selector),
			valueA:   valueA.String(),
			valueB:   "",
		})
}

func (d *differ) lineAB(mainSelector, selector string, valueA, valueB fmt.Stringer) {
	d.lines = append(
		d.lines,
		&patch{
			selector: joinSelectors(mainSelector, selector),
			valueA:   valueA.String(),
			valueB:   valueB.String(),
		})
}

func (d *differ) lineB(mainSelector, selector string, valueB fmt.Stringer) {
	d.lines = append(
		d.lines,
		&patch{
			selector: joinSelectors(mainSelector, selector),
			valueA:   "",
			valueB:   valueB.String(),
		})
}

func (d *differ) shouldCoerceNull(rules []*rule, selector string) bool {
    for _, rule := range rules {
        if rule.match(selector) && rule.action == coercenull {
            return true
        }
    }
    return false
}

func (d *differ) diffValues(selector string, valueA, valueB *objx.Value) error {

    // 1. coercion rules can solve nil case
    shouldCoreceA := d.shouldCoerceNull(d.rulesA, selector)
    shouldCoreceB := d.shouldCoerceNull(d.rulesB, selector)
    if  (shouldCoreceA && valueA.IsNil()) ||
        (shouldCoreceB && valueB.IsNil()) {
            return d.diffValuesCoerced(selector, valueA, valueB, shouldCoreceA, shouldCoreceB)
    }
    // 2. types mismatch
    if reflect.TypeOf(valueA.Data()) != reflect.TypeOf(valueB.Data()) {
        d.lineAB("", selector, valueA, valueB)
        return nil
    }

	switch {
	case valueA.IsInt():
        intA := valueA.MustInt()
        intB := valueB.MustInt()
		if intA != intB {
			d.lineAB("", selector, valueA, valueB)
		}
	case valueA.IsStr():
		strA := valueA.MustStr()
		strB := valueB.MustStr()
		if strA != strB {
			d.lineAB("", selector, valueA, valueB)
		}
	case valueA.IsInterSlice():
		err := d.diffInterSlice(selector, valueA, valueB)
		if err != nil {
			return err
		}
	case valueA.IsObjxMap():
		mA := valueA.MustObjxMap()
		mB := valueB.MustObjxMap()
		err := d.diffMap(selector, mA, mB)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Support for type %+v is not implemented", reflect.TypeOf(valueA.Data()))
	}
	return nil
}

// diffValuesCoerced allows an optional coercion of nulls
// TODO: join together with diffValues
func (d *differ) diffValuesCoerced(selector string, valueA, valueB *objx.Value, coerceA, coerceB bool) error {

    orNil := func(isType func(v *objx.Value)bool, valueA, valueB *objx.Value) bool {
        return (isType(valueA) || (coerceA && valueA.IsNil())) ||
               (isType(valueB) || (coerceB && valueB.IsNil()))
    }
    isInt := func(valueA, valueB *objx.Value) bool {
        isTyp := func(v *objx.Value) bool {return v.IsInt()}
        return orNil(isTyp, valueA, valueB)
    }
    isStr := func(valueA, valueB *objx.Value) bool {
        isTyp := func(v *objx.Value) bool {return v.IsStr()}
        return orNil(isTyp, valueA, valueB)
    }
    isInterSlice := func(valueA, valueB *objx.Value) bool {
        isTyp := func(v *objx.Value) bool {return v.IsInterSlice()}
        return orNil(isTyp, valueA, valueB)
    }
    isObjxMap := func(valueA, valueB *objx.Value) bool {
        isTyp := func(v *objx.Value) bool {return v.IsObjxMap()}
        return orNil(isTyp, valueA, valueB)
    }

	switch {
	case isInt(valueA, valueB):
        var intA, intB int
        if coerceA {
            intA = valueA.Int(0)
        } else {
            intA = valueA.MustInt()
        }
        if coerceB {
            intB = valueB.Int(0)
        } else {
            intB = valueB.MustInt()
        }
		if intA != intB {
			d.lineAB("", selector, valueA, valueB)
		}
	case isStr(valueA, valueB):
        var strA, strB string
        if coerceA {
            strA = valueA.Str("")
        } else {
            strA = valueA.MustStr()
        }
        if coerceB {
            strB = valueB.Str("")
        } else {
            strB = valueB.MustStr()
        }
		if strA != strB {
			d.lineAB("", selector, valueA, valueB)
		}
	case isInterSlice(valueA, valueB):
        if valueA.IsNil() {
            valueA = newValue([]interface{}{})
        }
        if valueB.IsNil() {
            valueB = newValue([]interface{}{})
        }
		err := d.diffInterSlice(selector, valueA, valueB)
		if err != nil {
			return err
		}
	case isObjxMap(valueA, valueB):
        var mA, mB objx.Map
        if coerceA {
            mA = valueA.ObjxMap(map[string]interface{}{})
        } else {
            mA = valueA.MustObjxMap()
        }
        if coerceB {
		    mB = valueB.ObjxMap(map[string]interface{}{})
        } else {
            mB = valueB.MustObjxMap()
        }
		err := d.diffMap(selector, mA, mB)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Support for type %+v is not implemented", reflect.TypeOf(valueA.Data()))
	}
	return nil
}

func newValue(i interface{}) *objx.Value {
	m := map[string]interface{}{
		"foo": i,
	}
	return objx.New(m).Get("foo")
}

func (d *differ) diffInterSlice(mainSelector string, valueA *objx.Value, valueB *objx.Value) error {
	if !valueA.IsInterSlice() || !valueA.IsInterSlice() {
		return fmt.Errorf("type mismatch for %s, valueA or valueB is not []interface{}, this is programming error", mainSelector)
	}
	iSliceA := valueA.MustInterSlice()
	iSliceB := valueB.MustInterSlice()
	for idx, a := range iSliceA {
		if len(iSliceB) <= idx {
			d.lineA(mainSelector, fmt.Sprintf("[%d]", idx), stringer{a})
			continue
		}
		b := iSliceB[idx]
		err := d.diffValues(joinSelectors(mainSelector, fmt.Sprintf("[%d]", idx)), newValue(a), newValue(b))
		if err != nil {
			return err
		}
	}

	if len(iSliceB) > len(iSliceA) {
		for idx := len(iSliceA); idx != len(iSliceB); idx++ {
			b := iSliceB[idx]
			d.lineB(mainSelector, fmt.Sprintf("[%d]", idx), stringer{b})
		}
	}
	return nil
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, len(m))
	idx := 0
	for key := range m {
		keys[idx] = key
		idx++
	}
	sort.Strings(keys)
	return keys
}

func (d *differ) diffMap(mainSelector string, objA objx.Map, objB objx.Map) error {

	visitedKeysA := make(map[string]struct{})
	for _, keyA := range sortedKeys(objA) {
		visitedKeysA[keyA] = struct{}{}
		valueA := objA.Get(keyA)
		// 1. objB missing data
		if !objB.Has(keyA) {
			d.lineA(mainSelector, keyA, valueA)
			continue
		}
		valueB := objB.Get(keyA)

		err := d.diffValues(joinSelectors(mainSelector, keyA), valueA, valueB)
		if err != nil {
			return err
		}
	}

	for _, keyB := range sortedKeys(objB) {
		if _, found := visitedKeysA[keyB]; found {
			continue
		}
		d.lineB(mainSelector, keyB, objB.Get(keyB))
	}

	return nil
}

func diff(jsA, jsB string) ([]*patch, error) {

	objA, err := objx.FromJSON(jsA)
	if err != nil {
		return []*patch{}, err
	}
	objB, err := objx.FromJSON(jsB)
	if err != nil {
		return []*patch{}, err
	}

	d := differ{
        lines: make([]*patch, 0, 64),
        rulesA: make([]*rule, 0),
        rulesB: make([]*rule, 0),
    }
	err = d.diffMap("", objA, objB)
	if err != nil {
		return d.lines, err
	}
	return d.lines, nil
}

func diff2(jsA, jsB string, rulesA, rulesB []*rule) ([]*patch, error) {

	objA, err := objx.FromJSON(jsA)
	if err != nil {
		return []*patch{}, err
	}
	objB, err := objx.FromJSON(jsB)
	if err != nil {
		return []*patch{}, err
	}

    if rulesA == nil {
        rulesA = make([]*rule, 0)
    }
    if rulesB == nil {
        rulesB = make([]*rule, 0)
    }

	d := differ{
        lines: make([]*patch, 0, 64),
        rulesA: rulesA,
        rulesB: rulesB,
    }
	err = d.diffMap("", objA, objB)
	if err != nil {
		return d.lines, err
	}
	return d.lines, nil
}
