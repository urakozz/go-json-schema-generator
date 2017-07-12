# go-json-schema-generator
[![codecov](https://codecov.io/gh/urakozz/go-json-schema-generator/branch/master/graph/badge.svg)](https://codecov.io/gh/urakozz/go-json-schema-generator)
[![travisci](https://travis-ci.org/urakozz/go-json-schema-generator.svg?branch=master)](https://travis-ci.org/urakozz/go-json-schema-generator)

Generate JSON Schema out of Golang schema

### Usage

First install the package
```
go get -u github.com/urakozz/go-json-schema-generator
```

Then create your generator file (see [Example](https://github.com/urakozz/go-json-schema-generator/blob/master/example) folder)
```
package main

import (
	"fmt"
	"github.com/urakozz/go-json-schema-generator"
)

type Domain struct {
	Data string `json:"data"`
}

func main(){
	fmt.Println(generator.Generate(&Domain{}))
}
```

### Supported tags

* `required:"true"` - field will be marked as required
* `description:"description"` - description will be added

On string fields:

* `minLength:"5"` - Set the minimum length of the value
* `maxLength:"5"` - Set the maximum length of the value
* `enum:"apple|banana|pear"` - Limit the available values to a defined set, separated by vertical bars
* `const:"I need to be there"` - Require the field to have a specific value.

On numeric types (strings and floats)

* `min:"-4.141592"` -  Set a minimum value
* `max:"123456789"` -  Set a maximum value
* `exclusiveMin:"0"` - Values must be strictly greater than this value
* `exclusiveMax:"11"` - Values must be strictly smaller than this value
* `const:"42"` - Property must have exactly this value.

### Expected behaviour

If struct field is pointer to the primitive type, then schema will allow this typa and null.
E.g.:

```
type Domain struct {
	NullableData *string `json:"nullableData"`
}
```
Output

```
{
    "$schema": "http://json-schema.org/schema#",
    "type": "object",
    "properties": {
        "nullableData": {
            "anyOf": [
                {
                    "type": "string"
                },
                {
                    "type": "null"
                }
            ]
        }
    }
}

```
