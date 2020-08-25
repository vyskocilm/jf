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

## Features

1. compare primitive values, ints, floats, bools and strings
2. compare arrays
3. compare other than root map
4. null coerce for A/B or both jsons
5. ignore certain keys
6. sort slice of maps by single in key
7. basic cmdline tool

## TODO

1. stringify numbers    ("42" == 42)
2. Go API (WIP)
3. rename keys
4. flags for cmdline tool (those starting by `x-` are temporary only and will be dropped)
5. better sorting support
6. ...

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


