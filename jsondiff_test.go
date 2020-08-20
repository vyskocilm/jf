package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSimpleMap tests diff handling of primitive types (int/string) and slices
func TestSimpleMap(t *testing.T) {

	const jsonA = `{
        "number": 42,
        "string": "hello",
        "strings": ["hello", "world"],
        "ints": [4, 2, 1]
    }`

	const jsonB = `{
        "number": 43,
        "string": "hellp",
        "strings": ["hello", "worle"],
        "ints": [4, 2, 99]
    }`

	assert := assert.New(t)
	lines, err := diff(jsonA, jsonB)
	assert.NoError(err)
	assert.Len(lines, 4)

    assert.Equal(&patch{"ints[2]", "1", "99"}, lines[0])
    assert.Equal(&patch{"number", "42", "43"}, lines[1])
    assert.Equal(&patch{"string", `"hello"`, `"hellp"`}, lines[2])
    assert.Equal(&patch{"strings[1]", `"world"`, `"worle"`}, lines[3])
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
    lines, err := diff(jsonA, jsonB)
    assert.NoError(err)
	assert.Len(lines, 2)

    assert.Equal(&patch{"numberA", "42", ""}, lines[0])
    assert.Equal(&patch{"numberB", "", "42"}, lines[1])
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
    lines, err := diff(jsonA, jsonB)
    assert.NoError(err)
	assert.Len(lines, 5)

    assert.Equal(&patch{"bigger[1]", "", "20"}, lines[0])
    assert.Equal(&patch{"bigger[2]", "", "30"}, lines[1])
    assert.Equal(&patch{"smaller[1]", "2", ""}, lines[2])
    assert.Equal(&patch{"weird[0]", "10", "30"}, lines[3])
    assert.Equal(&patch{"weird[1]", "20", "40"}, lines[4])
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
    lines, err := diff(jsonA, jsonB)
    assert.NoError(err)
	assert.Len(lines, 1)
    assert.Equal(&patch{"key.name", `"joe"`, `"Joe"`}, lines[0])
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
    lines, err := diff(jsonA, jsonB)
    assert.NoError(err)
	assert.Len(lines, 1)
    assert.Equal(&patch{"key.subkey.name", `"joe"`, `"Joe"`}, lines[0])
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
    lines, err := diff(jsonA, jsonB)
    assert.NoError(err)
	assert.Len(lines, 2)
    assert.Equal(&patch{"data[0].name", `"one"`, `"One"`}, lines[0])
    assert.Equal(&patch{"data[1].name", `"two"`, `"Two"`}, lines[1])
}

func TestNil(t *testing.T) {
    const jsonA = `{
        "key": null
    }`
    const jsonB = `{
        "key": 42
    }`

    assert := assert.New(t)
    lines, err := diff(jsonA, jsonB)
    assert.NoError(err)
	assert.Len(lines, 1)
    assert.Equal(&patch{"key", "null", "42"}, lines[0])
}

func TestCoerceNull(t *testing.T) {
    const jsonA = `{
        "key": null
    }`
    const jsonB = `{
        "key": 0
    }`

    coerceNull := []*rule {
        newCoerceNullRule("*"),
    }

    assert := assert.New(t)

    // 1. no coercion, return one line: see TestNil
    // 2. coercion of A, return 0 lines
    lines, err := diff2(jsonA, jsonB, coerceNull, nil)
    assert.NoError(err)
	assert.Len(lines, 0)
    // 3. coercion of B, return 1 line
    lines, err = diff2(jsonA, jsonB, nil, coerceNull)
    assert.NoError(err)
	assert.Len(lines, 1)
    assert.Equal(&patch{"key", "null", "0"}, lines[0])
    // 3. coercion of A/B, return 0 lines
    lines, err = diff2(jsonA, jsonB, coerceNull, coerceNull)
    assert.NoError(err)
	assert.Len(lines, 0)
}
