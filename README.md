[![CircleCI](https://circleci.com/gh/vyskocilm/jf.svg?style=svg)](https://circleci.com/gh/vyskocilm/jf) [![license](https://img.shields.io/badge/license-bsd--3--clause-green)](https://raw.githubusercontent.com/vyskocilm/jf/master/LICENSE)

# jf: Json difF

JSON diff library and simple CLI in Go. It can diff two arbitrary complex JSON
files if they starts as an json object.


## Installation and usage

```sh
git clone https://github.com/vyskocilm/jf
cd jf
go test
go build github.com/vyskocilm/jf/cmd/jf
./jf testdata/a.json testdata/b.json
bool       true    false
ints[2]    1       99
number     42      43
string     "hello" "hellp"
strings[1] "world" "worle"
```

**Warning**: depends on `reflect` and `gihub.com/stretchr/objx` and both MAY
panic in some circumstances.  Package `jf` type checks everything before use,
so it uses methods like `value.MustInt()`. However it panic itself if code ends
in an impossible (or not yet implemented one) situation. For example if
diffing code finds a weird type of `interface{}` or `*objx.Value` passed in,
like Go channel or a pointer. Those can't be passed in JSON.

In any case. Panic of `jf` is always a sign of a bug or missing feature, so do
not forget to create [an issue on GitHub](https://github.com/vyskocilm/jf/issues)
if you will find one.

## Features

1. compare primitive values, ints, floats, bools and strings
2. allows a specification of float equality function (can be used for int/float coercion)
3. compare arrays and maps
4. null coerce for A/B or both jsons
5. ignore certain keys
6. sort slice of maps by single id key (WIP: this shall be revised)
7. basic cmdline tool

## TODO

0. API docs
1. flags for cmdline tool (those starting by `x-` are temporary only and will be dropped)
2. better sorting support
3. custom comparator on objx.Value/objx.Value???

## Simple values
```
{
 "number": 42,
 "string": "hello",
 "strings": ["hello", "world"],
 "ints": [4, 2, 1]
}
------------------------------
{
 "number": 43,
 "string": "hellp",
 "strings": ["hello", "worle"],
 "ints": [4, 2, 99]
}
------------------------------
ints[2]    1       99
number     42      43
string     "hello" "hellp"
strings[1] "world" "worle"
```

## Nested maps

```
{
 "key": {
  "subkey": {
   "id": 11,
    "name": "joe"
  }
 }
}
------------------------------
{
 "key": {
  "subkey": {
   "id": 11,
    "name": "joe"
  }
 }
}
------------------------------
key.subkey.name "joe" "Joe"
```

See [jsondiff_test.go](jsondiff_test.go) for more examples.
