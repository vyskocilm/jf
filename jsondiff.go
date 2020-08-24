// jf: diff for JSON
package jf

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/stretchr/objx"
)

/*
   coercenull: make null == false, "", {} or [], 0
   ignore: ignore matching keys
   orderby: sort slice of maps
*/
type ruleAction int

const (
	coercenull ruleAction = iota
	ignore
	orderby
)

type jsoner interface {
	JSON() string
}

type jsonI struct {
	i interface{}
}

func (i jsonI) JSON() string {
	var inter interface{}
	switch v := i.i.(type) {
	case *objx.Value:
		inter = v.Data()
	default:
		inter = i.i
	}
	b, err := json.Marshal(inter)
	if err != nil {
		return fmt.Sprintf("%%!marshallError(%s)", err.Error())
	}
	return string(b)
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

type rule struct {
	selector    *regexp.Regexp
	action      ruleAction
	newKey      string
	excludedKey string
	orderByFunc func([]objx.Map)
}

type ruleDest int
const (
    RuleA ruleDest = iota
    RuleB
    RuleAB
)

// AddCoerceNull enables coercion of null value to empty value for given type
// "key": null will be equivalent of {}, [], "", 0 and false
func (d *Differ) AddCoerceNull(dest ruleDest, selector *regexp.Regexp) *Differ {
    return d.addRule(dest, &rule{selector: selector, action: coercenull})
}

func (d *Differ) addRule(dest ruleDest, rule *rule) *Differ {
    switch dest {
        case RuleA:
            d.rulesA = append(d.rulesA, rule)
        case RuleB:
            d.rulesB = append(d.rulesB, rule)
        case RuleAB:
            d.rulesA = append(d.rulesA, rule)
            d.rulesB = append(d.rulesB, rule)
    }
    return d
}

// AddIgnore adds selectors, which will be ignored in resulting diff
func (d *Differ) AddIgnore(dest ruleDest, selector *regexp.Regexp) *Differ {
    return d.addRule(dest, &rule{selector: selector, action: ignore})
}

// FIXME: can't specify the orderding rules
func newOrderbyKeyRule(selector *regexp.Regexp, key string) *rule {
    r := &rule{selector: selector, action: orderby}
	r.orderByFunc = func(msi []objx.Map) {
		sort.Slice(msi, func(i, j int) bool {
			intI := msi[i].Get(key).MustInt()
			intJ := msi[j].Get(key).MustInt()
			return intI < intJ
		})
	}
    return r
}

func (d *Differ) AddOrderByKey(dest ruleDest, selector *regexp.Regexp, key string) *Differ {
    return d.addRule(dest, newOrderbyKeyRule(selector, key))
}

func (r *rule) match(selector string) bool {
	return r.selector.MatchString(selector)
}

// SingleDiff express the difference of one JSON selector
type SingleDiff struct {
	selector string
	valueA   string
	valueB   string
}

func (d *SingleDiff) Selector() string {
    return d.selector
}

func (d *SingleDiff) A() string {
    return d.valueA
}

func (d *SingleDiff) B() string {
    return d.valueB
}

type DiffList []*SingleDiff
type rules []*rule

type Differ struct {
	diff  DiffList
	rulesA rules
	rulesB rules
}

func NewDiffer() *Differ {
    return &Differ{
		diff:  make(DiffList, 0, 64),
        rulesA: nil,
        rulesB: nil}
}

// lineA adds line with empty B value
func (d *Differ) lineA(mainSelector, selector string, valueA jsoner) {
	selector = joinSelectors(mainSelector, selector)
	shouldIgnoreA, _ := d.shouldIgnore(selector)
	if shouldIgnoreA {
		return
	}
	d.diff = append(
		d.diff,
		&SingleDiff{
			selector: selector,
			valueA:   valueA.JSON(),
			valueB:   "",
		})
}

func (d *Differ) lineAB(mainSelector, selector string, valueA, valueB jsoner) {
	selector = joinSelectors(mainSelector, selector)
	shouldIgnoreA, shouldIgnoreB := d.shouldIgnore(selector)
	if shouldIgnoreA || shouldIgnoreB {
		return
	}
	d.diff = append(
		d.diff,
		&SingleDiff{
			selector: selector,
			valueA:   valueA.JSON(),
			valueB:   valueB.JSON(),
		})
}

func (d *Differ) lineB(mainSelector, selector string, valueB jsoner) {
	selector = joinSelectors(mainSelector, selector)
	_, shouldIgnoreB := d.shouldIgnore(selector)
	if shouldIgnoreB {
		return
	}
	d.diff = append(
		d.diff,
		&SingleDiff{
			selector: selector,
			valueA:   "",
			valueB:   valueB.JSON(),
		})
}

func (d *Differ) matchRule(selector string, action ruleAction) (bool, bool) {
	matchA := false
	matchB := false
	for _, rule := range d.rulesA {
		if rule.action == action && rule.match(selector) {
			matchA = true
			break
		}
	}
	for _, rule := range d.rulesB {
		if rule.action == action && rule.match(selector) {
			matchB = true
			break
		}
	}
	return matchA, matchB
}

func (d *Differ) shouldCoerceNull(selector string) (bool, bool) {
	return d.matchRule(selector, coercenull)
}

func (d *Differ) shouldIgnore(selector string) (bool, bool) {
	return d.matchRule(selector, ignore)
}

func (d *Differ) orderByFuncs(selector string) ([]func([]objx.Map), []func([]objx.Map)) {
	retA := make([]func([]objx.Map), 0)
	for _, rule := range d.rulesA {
		if rule.action == orderby && rule.match(selector) {
			retA = append(retA, rule.orderByFunc)
		}
	}
	retB := make([]func([]objx.Map), 0)
	for _, rule := range d.rulesB {
		if rule.action == orderby && rule.match(selector) {
			retB = append(retB, rule.orderByFunc)
		}
	}
	return retA, retB
}

func (d *Differ) diffValues(selector string, valueA, valueB *objx.Value) error {

	// 1. coercion rules can solve nil case
	shouldCoerceA, shouldCoerceB := d.shouldCoerceNull(selector)
	if (shouldCoerceA && valueA.IsNil()) ||
		(shouldCoerceB && valueB.IsNil()) {
		return d.diffValuesCoerced(selector, valueA, valueB, shouldCoerceA, shouldCoerceB)
	}
	// 2. types mismatch
	if reflect.TypeOf(valueA.Data()) != reflect.TypeOf(valueB.Data()) {
		d.lineAB("", selector, jsonI{valueA}, jsonI{valueB})
		return nil
	}

	switch {
	case valueA.IsInt():
		intA := valueA.MustInt()
		intB := valueB.MustInt()
		if intA != intB {
			d.lineAB("", selector, jsonI{valueA}, jsonI{valueB})
		}
	case valueA.IsStr():
		strA := valueA.MustStr()
		strB := valueB.MustStr()
		if strA != strB {
			d.lineAB("", selector, jsonI{valueA}, jsonI{valueB})
		}
	case valueA.IsObjxMapSlice():
		err := d.diffObjxMapSlice(selector, valueA.MustObjxMapSlice(), valueB.MustObjxMapSlice())
		if err != nil {
			return err
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
func (d *Differ) diffValuesCoerced(selector string, valueA, valueB *objx.Value, coerceA, coerceB bool) error {

	orNil := func(isType func(v *objx.Value) bool, valueA, valueB *objx.Value) bool {
		return (isType(valueA) || (coerceA && valueA.IsNil())) ||
			(isType(valueB) || (coerceB && valueB.IsNil()))
	}
	isInt := func(valueA, valueB *objx.Value) bool {
		isTyp := func(v *objx.Value) bool { return v.IsInt() }
		return orNil(isTyp, valueA, valueB)
	}
	isStr := func(valueA, valueB *objx.Value) bool {
		isTyp := func(v *objx.Value) bool { return v.IsStr() }
		return orNil(isTyp, valueA, valueB)
	}
	isInterSlice := func(valueA, valueB *objx.Value) bool {
		isTyp := func(v *objx.Value) bool { return v.IsInterSlice() }
		return orNil(isTyp, valueA, valueB)
	}
	isObjxMap := func(valueA, valueB *objx.Value) bool {
		isTyp := func(v *objx.Value) bool { return v.IsObjxMap() }
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
			d.lineAB("", selector, jsonI{valueA}, jsonI{valueB})
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
			d.lineAB("", selector, jsonI{valueA}, jsonI{valueB})
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

func (d *Differ) diffInterSlice(mainSelector string, valueA *objx.Value, valueB *objx.Value) error {
	if !valueA.IsInterSlice() || !valueA.IsInterSlice() {
		return fmt.Errorf("type mismatch for %s, valueA or valueB is not []interface{}, this is programming error", mainSelector)
	}
	iSliceA := valueA.MustInterSlice()
	iSliceB := valueB.MustInterSlice()
	for idx, a := range iSliceA {
		if len(iSliceB) <= idx {
			d.lineA(mainSelector, fmt.Sprintf("[%d]", idx), jsonI{a})
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
			d.lineB(mainSelector, fmt.Sprintf("[%d]", idx), jsonI{b})
		}
	}
	return nil
}

func (d *Differ) diffObjxMapSlice(mainSelector string, sliceA, sliceB []objx.Map) error {

	orderByA, orderByB := d.orderByFuncs(mainSelector)
	for _, oba := range orderByA {
		oba(sliceA)
	}
	for _, obb := range orderByB {
		obb(sliceB)
	}

	for idx, a := range sliceA {
		if len(sliceB) <= idx {
			d.lineA(mainSelector, fmt.Sprintf("[%d]", idx), jsonI{a})
			continue
		}
		b := sliceB[idx]
		err := d.diffMap(joinSelectors(mainSelector, fmt.Sprintf("[%d]", idx)), a, b)
		if err != nil {
			return err
		}
	}

	if len(sliceB) > len(sliceA) {
		for idx := len(sliceA); idx != len(sliceB); idx++ {
			b := sliceB[idx]
			d.lineB(mainSelector, fmt.Sprintf("[%d]", idx), jsonI{b})
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

func (d *Differ) diffMap(mainSelector string, objA objx.Map, objB objx.Map) error {

	visitedKeysA := make(map[string]struct{})
	for _, keyA := range sortedKeys(objA) {
		visitedKeysA[keyA] = struct{}{}
		valueA := objA.Get(keyA)
		// 1. objB missing data
		if !objB.Has(keyA) {
			d.lineA(mainSelector, keyA, jsonI{valueA})
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
		d.lineB(mainSelector, keyB, jsonI{objB.Get(keyB)})
	}

	return nil
}

// Diff returns a list of individual list by comparing the jsonA and jsonB inputs
// supports arbitrary JSON files, if the top level is map of strings
func (d *Differ) Diff(jsonA, jsonB string) (DiffList, error) {

	objA, err := objx.FromJSON(jsonA)
	if err != nil {
		return []*SingleDiff{}, err
	}
	objB, err := objx.FromJSON(jsonB)
	if err != nil {
		return []*SingleDiff{}, err
	}

	d2 := Differ{
		diff:  make(DiffList, 0, 64),
		rulesA: d.rulesA,
		rulesB: d.rulesB,
	}
	err = d2.diffMap("", objA, objB)
	if err != nil {
		return d2.diff, err
	}
	return d2.diff, nil

}

func Diff(jsonA, jsonB string) (DiffList, error) {
    return NewDiffer().Diff(jsonA, jsonB)
}
