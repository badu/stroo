package stroo

import (
	"go/ast"
)

type TypeInfo struct {
	Package     string
	PackagePath string
	Name        string // in struct's case Name == Kind
	Kind        string // usually the way we Extract
	IsArray     bool   // if it's not array, it's struct
	IsPointer   bool   // if array, it's pointer
	IsImported  bool   // if array, it's pointer
	IsAlias     bool   // if array, it's pointer
	Fields      Fields
	MethodList  Methods
	comment     *ast.CommentGroup
	root        *Code // reference to the root document - to allow access to methods
}

func (t *TypeInfo) Root() *Code { return t.root }
func (t *TypeInfo) IsBasic() bool {
	return IsBasic(t.Kind)
}

type TypesSlice []*TypeInfo

// this should be called only from Code's StructByKey method
func (s TypesSlice) Extract(typeName string) *TypeInfo {
	for _, typeDef := range s {
		// first try by "Kind"
		if typeDef.Kind == typeName {
			return typeDef
		}
		// next try by "Name"
		if typeDef.Name == typeName {
			return typeDef
		}
	}
	return nil
}

func (s TypesSlice) ExtractType(typeName string) TypeInfo {
	for _, typeDef := range s {
		// first try by "Kind"
		if typeDef.Kind == typeName {
			return *typeDef
		}
		// next try by "Name"
		if typeDef.Name == typeName {
			return *typeDef
		}
	}
	return TypeInfo{}
}
