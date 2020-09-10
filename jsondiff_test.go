package jf

import (
	"math"
	"regexp"
	"testing"

	"github.com/stretchr/objx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func re(t *testing.T, s string) *regexp.Regexp {
	t.Helper()
	re, err := regexp.Compile(s)
	require.NoError(t, err)
	return re
}

// TestSimpleMap tests diff handling of primitive types (int/string) and slices
func TestSimpleMap(t *testing.T) {

	const jsonA = `{
        "number": 42,
        "string": "hello",
        "strings": ["hello", "world"],
        "ints": [4, 2, 1],
        "bool": true,
        "float": 11.1
    }`

	const jsonB = `{
        "number": 43,
        "string": "hellp",
        "strings": ["hello", "worle"],
        "ints": [4, 2, 99],
        "bool": false,
        "float": 11.11
    }`

	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 6)

	assert.Equal(SingleDiff{"bool", "true", "false"}, lines[0])
	assert.Equal(SingleDiff{"float", "11.1", "11.11"}, lines[1])
	assert.Equal(SingleDiff{"ints[2]", "1", "99"}, lines[2])
	assert.Equal(SingleDiff{"number", "42", "43"}, lines[3])
	assert.Equal(SingleDiff{"string", `"hello"`, `"hellp"`}, lines[4])
	assert.Equal(SingleDiff{"strings[1]", `"world"`, `"worle"`}, lines[5])
}

// TestDifferentKeys tests the case that in MSI there are different keys
func TestDifferentKeys(t *testing.T) {

	const jsonA = `{
        "numberA": 42
    }`
	const jsonB = `{
        "numberB": 42
    }`

	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 2)

	assert.Equal(SingleDiff{"numberA", "42", ""}, lines[0])
	assert.Equal(SingleDiff{"numberB", "", "42"}, lines[1])
}

// TestDifferentArrays sizes
func TestDifferentArrays(t *testing.T) {

	const jsonA = `{
        "smaller": [1, 2],
        "bigger": [1],
        "weird": [10, 20]
    }`
	const jsonB = `{
        "smaller": [1],
        "bigger": [1, 20, 30],
        "weird": [30, 40]
    }`

	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 5)

	assert.Equal(SingleDiff{"bigger[1]", "", "20"}, lines[0])
	assert.Equal(SingleDiff{"bigger[2]", "", "30"}, lines[1])
	assert.Equal(SingleDiff{"smaller[1]", "2", ""}, lines[2])
	assert.Equal(SingleDiff{"weird[0]", "10", "30"}, lines[3])
	assert.Equal(SingleDiff{"weird[1]", "20", "40"}, lines[4])
}

func TestMapInMap(t *testing.T) {
	const jsonA = `{
        "key": {
            "id": 11,
            "name": "joe"
        }
    }`
	const jsonB = `{
        "key": {
            "id": 11,
            "name": "Joe"
        }
    }`

	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 1)
	assert.Equal(SingleDiff{"key.name", `"joe"`, `"Joe"`}, lines[0])
}

func TestMapInMapInMap(t *testing.T) {
	const jsonA = `{
        "key": {
            "subkey": {
                "id": 11,
                "name": "joe"
            }
        }
    }`
	const jsonB = `{
        "key": {
            "subkey": {
                "id": 11,
                "name": "Joe"
            }
        }
    }`

	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 1)
	assert.Equal(SingleDiff{"key.subkey.name", `"joe"`, `"Joe"`}, lines[0])
}

func TestMapSlice(t *testing.T) {
	const jsonA = `{
        "data": [
            {"id": 1, "name": "one"},
            {"id": 2, "name": "two"}
        ]
    }
    `
	const jsonB = `{
        "data": [
            {"id": 1, "name": "One"},
            {"id": 2, "name": "Two"}
        ]
    }
    `

	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 2)
	assert.Equal(SingleDiff{"data[0].name", `"one"`, `"One"`}, lines[0])
	assert.Equal(SingleDiff{"data[1].name", `"two"`, `"Two"`}, lines[1])
}

func TestNil(t *testing.T) {
	const jsonA = `{
        "key": null
    }`
	const jsonB = `{
        "key": 42
    }`

	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 1)
	assert.Equal(SingleDiff{"key", "null", "42"}, lines[0])
}

// TestCoerceNull tests null coercion of jsonA only, jsonB only and both
func TestCoerceNull(t *testing.T) {
	const jsonA = `{
        "key": null
    }`
	const jsonB = `{
        "key": 0
    }`

	assert := assert.New(t)

	// 1. no coercion, return one line: see TestNil
	// 2. coercion of A, return 0 lines
	lines, err := NewDiffer().AddCoerceNull(RuleA, re(t, `.*`)).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
	// 3. coercion of B, return 1 line
	lines, err = NewDiffer().AddCoerceNull(RuleB, re(t, `.*`)).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 1)
	assert.Equal(SingleDiff{"key", "null", "0"}, lines[0])
	// 3. coercion of A/B, return 0 lines
	lines, err = NewDiffer().AddCoerceNull(RuleAB, re(t, `.*`)).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}

// TestCoerceNullMatch tests that regexp based match works
func TestCoerceNullMatch(t *testing.T) {
	const jsonA = `{
        "key": {
            "subkey1": null,
            "subkey2": null
        }
    }`
	const jsonB = `{
        "key": {
            "subkey1": 0,
            "subkey2": 0
        }
    }`
	assert := assert.New(t)

	lines, err := NewDiffer().AddCoerceNull(RuleA, re(t, `key\.subkey1`)).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 1)
	assert.Equal(SingleDiff{"key.subkey2", "null", "0"}, lines[0])
}

func TestIgnore(t *testing.T) {
	const jsonA = `{
        "id": 11
    }`
	const jsonB = `{
        "id": 11,
        "additional": 42
    }`
	assert := assert.New(t)

	lines, err := NewDiffer().AddIgnore(RuleA, re(t, `additional`)).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 1)
	assert.Equal(SingleDiff{"additional", "", "42"}, lines[0])

	lines, err = NewDiffer().AddIgnore(RuleB, re(t, `additional`)).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}

func TestIgnoreZero(t *testing.T) {

	const jsonA = `{
        "number": 0,
        "string": "",
        "strings": [],
        "objects": {},
        "bool": false
    }`

	const jsonB = `{
    }`

	assert := assert.New(t)
	lines, err := NewDiffer().
		AddIgnoreIfZero(RuleA, re(t, "number")).
		AddIgnoreIfZero(RuleA, re(t, "string")).
		AddIgnoreIfZero(RuleA, re(t, "strings")).
		AddIgnoreIfZero(RuleA, re(t, "objects")).
		AddIgnoreIfZero(RuleA, re(t, "bool")).
		Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}

// test custom float function
func TestFloatEqual(t *testing.T) {
	const jsonA = `{
        "float": 1234.003,
        "floats": [1234.003],
        "floatm": {"first": 1234.003}
    }
    `
	const jsonB = `{
        "float": 1234.1,
        "floats": [1234.1],
        "floatm": {"first": 1234.1}
    }
    `

	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 3)

	eq := func(a, b float64) bool {
		abs := math.Abs(a - b)
		ret := abs <= 0.5
		return ret
	}

	lines, err = NewDiffer().AddFloatEqual(re(t, ".*"), eq).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}

// test int to float coercion
func TestFloatIntEqual(t *testing.T) {
	const jsonA = `{
        "float": 1234.003,
        "floats": [1234.003],
        "floatm": {"first": 1234.003}
    }
    `
	const jsonB = `{
        "float": 1234,
        "floats": [1234],
        "floatm": {"first": 1234}
    }
    `
	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 3)

	eq := func(a, b float64) bool {
		abs := math.Abs(a - b)
		ret := abs <= 1.0
		return ret
	}

	lines, err = NewDiffer().AddFloatEqual(re(t, ".*"), eq).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}

// test ignore order for slices
func TestIgnoreOrder(t *testing.T) {
	const jsonA = `{
        "list": [1, 2, 3]
    }
    `
	const jsonB = `{
        "list": [3, 2, 1]
    }
    `
	assert := assert.New(t)
	lines, err := NewDiffer().AddIgnoreOrder(re(t, "list")).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}

// test number equal number
func TestStringNumber(t *testing.T) {
	const jsonA = `{
        "int": "1",
        "float": "11.11",
        "notnumber": "a"
    }
    `
	const jsonB = `{
        "int": 1,
        "float": 11.11,
        "notnumber": "a"
    }
    `
	assert := assert.New(t)
	lines, err := NewDiffer().AddStringNumber(re(t, ".*")).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}

// test number equal number
func TestCustomEqual(t *testing.T) {
	const jsonA = `{
        "int": 1110
    }
    `
	const jsonB = `{
        "int": 1120
    }
    `

	// return true only if both are numbers and difference is less than 10
	// artifical example as I know the types of items
	eq := func(selector string, a, b *objx.Value) bool {
		if selector != "int" {
			return false
		}
		if !a.IsInt() || !b.IsInt() {
			return false
		}
		intA := a.MustInt()
		intB := a.MustInt()
		diff := intA - intB
		if diff < 0 {
			diff *= -1
		}

		return diff < 20
	}

	assert := assert.New(t)
	lines, err := NewDiffer().AddCustomEqual(re(t, ".*"), eq).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}
