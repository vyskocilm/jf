// jf provides diffing of arbitrary jsons
//
// The result of a diff is a
//
//      type SingleDiff struct {
//	        selector string
//    	    valueA   string
//    	    valueB   string
//      }
//
// Where selector is JSON path selector describing the place where the
// difference was found. For example consider following inputs
//
//      const a=`{"data": {"key": "foo"}}`
//      const b=`{"data": {"key": "bar"}}`
//
//      diff, err := Diff(a, b)
//
//      // diff[0]
//      SingleDiff{selector: "data.key", valueA: "foo", valueB: "bar"}
//
// jf does exact diffing by default, but can be instructed to coerce or ignore
// certain part of JSON.
//
// To deal with a real world complexities, jf can skip or transform parts of
// JSONs during the transformation. The Differ can save the transformation
// rules, which will change the jf behavior.
//
//	const a = `{"list": [1, 2, 3]}`
//  const jsonB = `{"list": [3, 2, 1]}`
//	re := func(s string) *regexp.Regexp { return regexp.MustCompile(s) }
//
//	lines, err := NewDiffer().AddIgnoreOrder(re("list")).Diff(jsonA, jsonB)
//
//  len(lines)
//  0
//
//  Each key is regexp matched to json PATH of a current element, so "list"
//  means apply to ANY path, with string list inside.
//

package jf

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/stretchr/objx"
)

/*
   coercenull: make null == false, "", {} or [], 0
   ignore: ignore matching keys
   ignoreIfZero: ignores matching keys if value is zero (false, "", 0, [] or {})
   floatEqual: adds function for comparing floats
   ignoreOrder: ignore order of arrays (nop for other types)
   stringnumber: make "1" equal 1
*/
type ruleAction int

const (
	coercenull ruleAction = iota
	ignore
	ignoreIfZero
	floatEqual
	ignoreOrder
	stringNumber
)

// FloatEqualFn is a function comparing two floats
type FloatEqualFunc func(float64, float64) bool

type jsoner interface {
	JSON() string
	isZero() bool
}

type jsonI struct {
	i              interface{}
	floatEqualFunc FloatEqualFunc
}

func defaultFloatEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
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

func (i jsonI) isZero() bool {
	switch v := i.i.(type) {
	case *objx.Value:
		switch {
		case v.IsInt():
			return v.MustInt() == 0
		case v.IsFloat64():
			return i.floatEqualFunc(v.MustFloat64(), 0.0)
		case v.IsStr():
			return v.MustStr() == ""
		case v.IsBool():
			return !v.MustBool()
		case v.IsInterSlice():
			return len(v.MustInterSlice()) == 0
		case v.IsObjxMap():
			return len(v.MustObjxMap()) == 0
		case v.IsNil():
			return true
		default:
			errmsg := fmt.Errorf("isZero does not support *objx.Value %+v", v.Data())
			panic(errmsg)
		}
	case int:
		return v == 0
	case float64:
		return i.floatEqualFunc(v, 0.0)
	case string:
		return v == ""
	case bool:
		return !v
	}
	errmsg := fmt.Errorf("isZero does not support %T %+v", i, i)
	panic(errmsg)
}

func (i jsonI) String() string {
	return fmt.Sprintf("%T %+v", i.i, i.i)
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
	selector       *regexp.Regexp
	action         ruleAction
	floatEqualFunc FloatEqualFunc
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

// AddIgnoreIfEmpty adds selectors, which will be ignored in a case value is empty
func (d *Differ) AddIgnoreIfZero(dest ruleDest, selector *regexp.Regexp) *Differ {
	return d.addRule(dest, &rule{selector: selector, action: ignoreIfZero})
}

// AddFloatEqual adds a function for comparing floats. Default function expects
// numbers to be the same up to 1e-9
func (d *Differ) AddFloatEqual(selector *regexp.Regexp, fn FloatEqualFunc) *Differ {
	return d.addRule(RuleAB, &rule{selector: selector, action: floatEqual, floatEqualFunc: fn})
}

// AddIgnoreRule ignores order of arrays, so [1, 2, 3] == [3, 2, 1]
func (d *Differ) AddIgnoreOrder(selector *regexp.Regexp) *Differ {
	return d.addRule(RuleAB, &rule{selector: selector, action: ignoreOrder})
}

// AddStringNumber equals "1" == 1
func (d *Differ) AddStringNumber(selector *regexp.Regexp) *Differ {
	return d.addRule(RuleAB, &rule{selector: selector, action: stringNumber})
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

type DiffList []SingleDiff
type rules []*rule

// Differ traverse through JSONS and diff each part. It stores actual
// differences and can apply rules to different parts for comparison
type Differ struct {
	diff   DiffList
	rulesA rules
	rulesB rules
}

// NewDiffer creates new empty differ with no rules. It can get additional
// processing rules via AddXYZ methods
func NewDiffer() *Differ {
	return &Differ{
		diff:   make(DiffList, 0, 64),
		rulesA: nil,
		rulesB: nil}
}

// clone creates an empty Differ with the same set of rules
func (d *Differ) clone() *Differ {
	return &Differ{
		diff:   make(DiffList, 0, 64),
		rulesA: d.rulesA,
		rulesB: d.rulesB,
	}
}

// lineA adds line with empty B value
func (d *Differ) lineA(mainSelector, selector string, valueA jsoner) {
	selector = joinSelectors(mainSelector, selector)
	shouldIgnoreA, _ := d.shouldIgnore(selector)
	if shouldIgnoreA {
		return
	}
	shouldIgnoreIfZeroA, _ := d.shouldIgnoreIfZero(selector)
	if shouldIgnoreIfZeroA && valueA.isZero() {
		return
	}
	d.diff = append(
		d.diff,
		SingleDiff{
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
	shouldIgnoreIfZeroA, shouldIgnoreIfZeroB := d.shouldIgnoreIfZero(selector)
	if (shouldIgnoreIfZeroA && valueA.isZero()) || (shouldIgnoreIfZeroB && valueB.isZero()) {
		return
	}
	d.diff = append(
		d.diff,
		SingleDiff{
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
	_, shouldIgnoreIfZeroB := d.shouldIgnoreIfZero(selector)
	if shouldIgnoreIfZeroB && valueB.isZero() {
		return
	}
	d.diff = append(
		d.diff,
		SingleDiff{
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

func (d *Differ) shouldIgnoreIfZero(selector string) (bool, bool) {
	return d.matchRule(selector, ignoreIfZero)
}

func (d *Differ) floatEqualFunc(selector string) FloatEqualFunc {
	for _, rule := range d.rulesA {
		if rule.action == floatEqual && rule.match(selector) {
			return rule.floatEqualFunc
		}
	}
	return defaultFloatEqual
}

func (d *Differ) shouldIgnoreOrder(selector string) bool {
	a, _ := d.matchRule(selector, ignoreOrder)
	return a
}

func (d *Differ) shouldConvertStringToNumber(selector string) bool {
	a, _ := d.matchRule(selector, stringNumber)
	return a
}

// mustFloat64 unpack int of float64 as float64
func mustFloat64(v *objx.Value) float64 {
	if v.IsInt() {
		return float64(v.MustInt())
	}
	return v.MustFloat64()
}

// float64 returns number or zero, coerce int to float64
func float64OrZero(v *objx.Value) float64 {
	if v.IsInt() {
		return float64(v.Int(0))
	}
	return v.Float64(0.0)
}

// tryAsNumber parse "1" or "1.1" as int/float64 and return *objx.Value
// with a proper type. If it can't be parsed, it returns original value
func tryAsNumber(valueA *objx.Value) *objx.Value {
	if !valueA.IsStr() {
		return valueA
	}
	aStr := valueA.MustStr()
	isNumber := false
	if strings.Contains(aStr, ".") {
		_, err := strconv.ParseFloat(aStr, 64)
		isNumber = err == nil
	} else {
		_, err := strconv.Atoi(aStr)
		isNumber = err == nil
	}

	if !isNumber {
		return valueA
	}
	return objx.MustFromJSON(fmt.Sprintf(`{"a": %s}`, aStr)).Get("a")
}

func (d *Differ) diffValues(selector string, valueA, valueB *objx.Value) error {

	// 1. coercion rules can solve nil case
	shouldCoerceA, shouldCoerceB := d.shouldCoerceNull(selector)
	if (shouldCoerceA && valueA.IsNil()) ||
		(shouldCoerceB && valueB.IsNil()) {
		return d.diffValuesCoerced(selector, valueA, valueB, shouldCoerceA, shouldCoerceB)
	}

	// try to parse number as string to number
	if d.shouldConvertStringToNumber(selector) {
		valueA = tryAsNumber(valueA)
		valueB = tryAsNumber(valueB)
	}

	// coerce ints and floats by default
	if (valueA.IsInt() || valueA.IsFloat64()) &&
		(valueB.IsInt() || valueB.IsFloat64()) {
		goto skipTypeCheck
	}

	// 2. types mismatch
	if reflect.TypeOf(valueA.Data()) != reflect.TypeOf(valueB.Data()) {
		d.lineAB("", selector, jsonI{i: valueA}, jsonI{i: valueB})
		return nil
	}

skipTypeCheck:
	switch {
	case valueA.IsFloat64() || valueB.IsFloat64():
		floatA := mustFloat64(valueA)
		floatB := mustFloat64(valueB)
		floatEqualFunc := d.floatEqualFunc(selector)
		if !floatEqualFunc(floatA, floatB) {
			d.lineAB("", selector, jsonI{valueA, floatEqualFunc}, jsonI{valueB, floatEqualFunc})
		}
	case valueA.IsBool() && valueB.IsBool():
		intA := valueA.MustBool()
		intB := valueB.MustBool()
		if intA != intB {
			d.lineAB("", selector, jsonI{i: valueA}, jsonI{i: valueB})
		}
	case valueA.IsInt():
		intA := valueA.MustInt()
		intB := valueB.MustInt()
		if intA != intB {
			d.lineAB("", selector, jsonI{i: valueA}, jsonI{i: valueB})
		}
	case valueA.IsStr():
		strA := valueA.MustStr()
		strB := valueB.MustStr()
		if strA != strB {
			d.lineAB("", selector, jsonI{i: valueA}, jsonI{i: valueB})
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
		return (isType(valueA) || (coerceA && valueA.IsNil())) &&
			(isType(valueB) || (coerceB && valueB.IsNil()))
	}
	isBool := func(valueA, valueB *objx.Value) bool {
		isTyp := func(v *objx.Value) bool { return v.IsBool() }
		return orNil(isTyp, valueA, valueB)
	}
	isInt := func(valueA, valueB *objx.Value) bool {
		isTyp := func(v *objx.Value) bool { return v.IsInt() }
		return orNil(isTyp, valueA, valueB)
	}
	isFloat64 := func(valueA, valueB *objx.Value) bool {
		// simply - either valueA or valueB must be float64
		//          and if so, then valueA/valueB can be one of float64/int/null
		// iow isFloat64(int, int) returns false
		return (valueA.IsFloat64() || valueB.IsFloat64()) &&
			((valueA.IsFloat64() || valueA.IsInt() || (coerceA && valueA.IsNil())) &&
				(valueB.IsFloat64() || valueB.IsInt() || (coerceB && valueB.IsNil())))
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
	case isFloat64(valueA, valueB):
		var floatA, floatB float64
		if coerceA {
			floatA = float64OrZero(valueA)
		} else {
			floatA = mustFloat64(valueA)
		}
		if coerceB {
			floatB = float64OrZero(valueB)
		} else {
			floatB = mustFloat64(valueB)
		}
		floatEqualFunc := d.floatEqualFunc(selector)
		if !floatEqualFunc(floatA, floatB) {
			d.lineAB("", selector, jsonI{valueA, floatEqualFunc}, jsonI{valueB, floatEqualFunc})
		}
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
			d.lineAB("", selector, jsonI{i: valueA}, jsonI{i: valueB})
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
			d.lineAB("", selector, jsonI{i: valueA}, jsonI{i: valueB})
		}
	case isBool(valueA, valueB):
		//XXX: isBool check must be after isInt (and probably isStr) otherwise
		//  A={"key": null}, B={"key": 0} with null corecion will fail
		var intA, intB bool
		if coerceA {
			intA = valueA.Bool(false)
		} else {
			intA = valueA.MustBool()
		}
		if coerceB {
			intB = valueB.Bool(false)
		} else {
			intB = valueB.MustBool()
		}
		if intA != intB {
			d.lineAB("", selector, jsonI{i: valueA}, jsonI{i: valueB})
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
	if !valueA.IsInterSlice() || !valueB.IsInterSlice() {
		return fmt.Errorf("type mismatch for %s, valueA or valueB is not []interface{}, this is programming error", mainSelector)
	}

	if d.shouldIgnoreOrder(mainSelector) {
		equal, err := d.diffInterSliceDetectEquals(mainSelector, valueA, valueB)
		if err != nil {
			return err
		}
		if equal {
			return nil
		}
	}

	iSliceA := valueA.MustInterSlice()
	iSliceB := valueB.MustInterSlice()
	for idx, a := range iSliceA {
		if len(iSliceB) <= idx {
			selector := joinSelectors(mainSelector, fmt.Sprintf("[%d]", idx))
			floatEqualFunc := d.floatEqualFunc(selector)
			d.lineA("", selector, jsonI{i: a, floatEqualFunc: floatEqualFunc})
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
			selector := joinSelectors(mainSelector, fmt.Sprintf("[%d]", idx))
			floatEqualFunc := d.floatEqualFunc(selector)
			d.lineB("", selector, jsonI{i: b, floatEqualFunc: floatEqualFunc})
		}
	}
	return nil
}

type intSet map[int]struct{}

func newIntSet() intSet {
	return make(map[int]struct{})
}

func (s intSet) Add(v int) {
	s[v] = struct{}{}
}

func (s intSet) Has(v int) bool {
	_, has := s[v]
	return has
}

func (d *Differ) diffInterSliceDetectEquals(mainSelector string, valueA *objx.Value, valueB *objx.Value) (bool, error) {

	idxEqualA := newIntSet()
	idxEqualB := newIntSet()

	if !valueA.IsInterSlice() || !valueB.IsInterSlice() {
		return false, fmt.Errorf("type mismatch for %s, valueA or valueB is not []interface{}, this is programming error", mainSelector)
	}

	iSliceA := valueA.MustInterSlice()
	iSliceB := valueB.MustInterSlice()

	// clone Differ with empty diff - this help code reuse and won't mess with
	// the main diff
	other := d.clone()
	for idxA, a := range iSliceA {
		for idxB, b := range iSliceB {
			if idxEqualB.Has(idxB) {
				continue
			}

			other.diff = make(DiffList, 0, 64)
			err := other.diffValues(joinSelectors(mainSelector, fmt.Sprintf("[%d]", idxA)), newValue(a), newValue(b))
			if err != nil {
				return false, err
			}
			if len(other.diff) == 0 {
				idxEqualA.Add(idxA)
				idxEqualB.Add(idxB)
				break
			}
		}
		idxEqualA.Add(idxA)
	}

	return len(idxEqualA) == len(iSliceA) && len(idxEqualB) == len(iSliceB), nil
}

func (d *Differ) diffObjxMapSlice(mainSelector string, sliceA, sliceB []objx.Map) error {

	for idx, a := range sliceA {
		if len(sliceB) <= idx {
			selector := joinSelectors(mainSelector, fmt.Sprintf("[%d]", idx))
			floatEqualFunc := d.floatEqualFunc(selector)
			d.lineA("", selector, jsonI{i: a, floatEqualFunc: floatEqualFunc})
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
			selector := joinSelectors(mainSelector, fmt.Sprintf("[%d]", idx))
			floatEqualFunc := d.floatEqualFunc(selector)
			d.lineB("", selector, jsonI{i: b, floatEqualFunc: floatEqualFunc})
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
			selector := joinSelectors(mainSelector, keyA)
			floatEqualFunc := d.floatEqualFunc(selector)
			d.lineA("", selector, jsonI{i: valueA, floatEqualFunc: floatEqualFunc})
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
		selector := joinSelectors(mainSelector, keyB)
		floatEqualFunc := d.floatEqualFunc(selector)
		d.lineB("", selector, jsonI{i: objB.Get(keyB), floatEqualFunc: floatEqualFunc})
	}

	return nil
}

// Diff returns a list of individual differences by comparing the jsonA and
// jsonB inputs supports arbitrary JSON files, if the top level is map of
// strings
func (d *Differ) Diff(jsonA, jsonB string) (DiffList, error) {

	objA, err := objx.FromJSON(jsonA)
	if err != nil {
		return []SingleDiff{}, err
	}
	objB, err := objx.FromJSON(jsonB)
	if err != nil {
		return []SingleDiff{}, err
	}

	d2 := Differ{
		diff:   make(DiffList, 0, 64),
		rulesA: d.rulesA,
		rulesB: d.rulesB,
	}
	err = d2.diffMap("", objA, objB)
	if err != nil {
		return d2.diff, err
	}
	return d2.diff, nil

}

// Diff is a shortcut for NewDiffer().Diff, exact diffing without any filters
func Diff(jsonA, jsonB string) (DiffList, error) {
	return NewDiffer().Diff(jsonA, jsonB)
}
