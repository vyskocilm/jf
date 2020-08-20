# jf: Json difF

Compare arbitrary nested JSONS key by key. Prints an output in tabular format
`JSONPATH jsonA jsonB`, so it can be trivially handled by unix textutils.


## Installation and usage

```sh
git clone https://github.com/vyskocilm/jf
cd jf
go test
go build
./jf testdata/a.json testdata/b.json 
ints[2]    1       99
number     42      43
string     "hello" "hellp"
strings[1] "world" "worle"
```

## Features

1. compare primitive values, ints and strings
2. compare arrays
3. compare other than root map
4. null coerce for A/B or both jsons
5. ignore certain keys
6. sort slice of maps by single in key
7. basic cmdline tool

## TODO

1. support for floats
2. stringify numbers    ("42" == 42)
3. Go API
4. rename keys
5. flags for cmdline tool (those starting by `x-` are temporary only and will be dropped)
6. better sorting support

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
