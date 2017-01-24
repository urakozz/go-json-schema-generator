# go-json-schema-generator
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
