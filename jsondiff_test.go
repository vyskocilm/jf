package jf

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSimpleMap tests diff handling of primitive types (int/string) and slices
func TestSimpleMap(t *testing.T) {

	const jsonA = `{
        "number": 42,
        "string": "hello",
        "strings": ["hello", "world"],
        "ints": [4, 2, 1],
        "bool": true
    }`

	const jsonB = `{
        "number": 43,
        "string": "hellp",
        "strings": ["hello", "worle"],
        "ints": [4, 2, 99],
        "bool": false
    }`

	assert := assert.New(t)
	lines, err := Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 5)

	assert.Equal(&SingleDiff{"bool", "true", "false"}, lines[0])
	assert.Equal(&SingleDiff{"ints[2]", "1", "99"}, lines[1])
	assert.Equal(&SingleDiff{"number", "42", "43"}, lines[2])
	assert.Equal(&SingleDiff{"string", `"hello"`, `"hellp"`}, lines[3])
	assert.Equal(&SingleDiff{"strings[1]", `"world"`, `"worle"`}, lines[4])
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

	assert.Equal(&SingleDiff{"numberA", "42", ""}, lines[0])
	assert.Equal(&SingleDiff{"numberB", "", "42"}, lines[1])
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

	assert.Equal(&SingleDiff{"bigger[1]", "", "20"}, lines[0])
	assert.Equal(&SingleDiff{"bigger[2]", "", "30"}, lines[1])
	assert.Equal(&SingleDiff{"smaller[1]", "2", ""}, lines[2])
	assert.Equal(&SingleDiff{"weird[0]", "10", "30"}, lines[3])
	assert.Equal(&SingleDiff{"weird[1]", "20", "40"}, lines[4])
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
	assert.Equal(&SingleDiff{"key.name", `"joe"`, `"Joe"`}, lines[0])
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
	assert.Equal(&SingleDiff{"key.subkey.name", `"joe"`, `"Joe"`}, lines[0])
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
	assert.Equal(&SingleDiff{"data[0].name", `"one"`, `"One"`}, lines[0])
	assert.Equal(&SingleDiff{"data[1].name", `"two"`, `"Two"`}, lines[1])
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
	assert.Equal(&SingleDiff{"key", "null", "42"}, lines[0])
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
	dotStar, err := regexp.Compile(".*")
	assert.NoError(err)

	// 1. no coercion, return one line: see TestNil
	// 2. coercion of A, return 0 lines
	lines, err := NewDiffer().AddCoerceNull(RuleA, dotStar).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
	// 3. coercion of B, return 1 line
	lines, err = NewDiffer().AddCoerceNull(RuleB, dotStar).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 1)
	assert.Equal(&SingleDiff{"key", "null", "0"}, lines[0])
	// 3. coercion of A/B, return 0 lines
	lines, err = NewDiffer().AddCoerceNull(RuleAB, dotStar).Diff(jsonA, jsonB)
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

	keyDotSubkey1, err := regexp.Compile("key\\.subkey1")
	assert.NoError(err)

	lines, err := NewDiffer().AddCoerceNull(RuleA, keyDotSubkey1).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 1)
	assert.Equal(&SingleDiff{"key.subkey2", "null", "0"}, lines[0])
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

	additional, err := regexp.Compile("additional")
	assert.NoError(err)

	lines, err := NewDiffer().AddIgnore(RuleA, additional).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 1)
	assert.Equal(&SingleDiff{"additional", "", "42"}, lines[0])

	lines, err = NewDiffer().AddIgnore(RuleB, additional).Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}

func TestSort(t *testing.T) {
	const jsonA = `{
        "data": [
            {"id": 11, "name": "eleven"},
            {"id": 1, "name": "one"},
            {"id": 7, "name": "seven"}
        ]
    }`
	const jsonB = `{
        "data": [
            {"id": 7, "name": "seven"},
            {"id": 11, "name": "eleven"},
            {"id": 1, "name": "one"}
        ]
    }`

	assert := assert.New(t)
	data, err := regexp.Compile("data")
	assert.NoError(err)

	lines, err := NewDiffer().AddOrderByKey(RuleA, data, "id").Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 6)

	lines, err = NewDiffer().AddOrderByKey(RuleAB, data, "id").Diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 0)
}
