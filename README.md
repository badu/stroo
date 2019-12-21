# stroo

Having the following declaration:

```go
package model
//go:generate stroo -type=SomeJsonPayload -output=model_json_gen.go -template=./../../templates/json_marshal.tmpl
//go:generate stroo -type=SomeJsonPayload -output=model_json_gen.go -template=./../../templates/json_unmarshal.tmpl
type SomeJsonPayload struct{
	Name string `json:"name"`
}
```

stroo will use the template (relative path in the example `json_marshal.tmpl` and `json_unmarshal.tmpl`) to generate the 
files indicated as output, in the same package with the struct declaration.

Currently, this is work in progress - the proof of concept is under heavy construction.

Code scan folder contains code taken (and modified) from [tools](golang.org/x/tools) - static analysis part. Thank you good authors!

### Notes

Store and retrieve from template : templates can store and retrieve key-values by using {{ .Store <key> <value> }} and retrieve them with {{ .Retrieve <key> }} where <key> is a `string` and <value> is `interface{}`.
