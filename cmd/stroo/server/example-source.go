// This is the default "source" to help you build your template.
// Modify it as you wish.
package server

type TestInterfaceOne interface {
	AMethod()
}

type TestStruct struct {
	Field1 int
	Field2 bool
	Field3 string
}

func TestFunc(a, b int) (bool, error) {
	return true, nil
}
