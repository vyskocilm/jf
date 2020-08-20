# Jf (Json difF)

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
6. basic cmdline tool

## TODO

1. support for floats
2. stringify numbers    ("42" == 42)
3. better matchine algorithm for rules
4. Go API
5. rename keys
6. flags for cmdline tool

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
