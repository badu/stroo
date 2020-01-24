package stroo

import (
	"go/ast"
)

type TypeInfo struct {
	Package     string
	PackagePath string
	Name        string
	Kind        string
	IsArray     bool // if it's not array, it's struct
	IsPointer   bool // if array, it's pointer
	Fields      Fields
	MethodList  Methods
	Comment     *ast.CommentGroup
	root        *Code // reference to the root document - to allow access to methods
}

func (t *TypeInfo) Root() *Code { return t.root }

type TypesSlice []*TypeInfo

// this should be called only from Code's StructByKey method
func (s TypesSlice) Extract(typeName string) *TypeInfo {
	for _, typeDef := range s {
		if typeDef.Name == typeName {
			return typeDef
		}
	}
	return nil
}
