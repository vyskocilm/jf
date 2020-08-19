# Jf (Json difF)

Compare arbitrary nested JSONS key by key.

## Features

1. compare primitive values, ints and strings
2. compare arrays
3. compare other than root map
4. null coerce for A/B or both jsons

## TODO

1. support for floats
2. stringify numbers    ("42" == 42)
3. exclude or rename keys
4. better matchine algorithm for rules
5. Go API
6. cmdline tool

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
