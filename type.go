package stroo

import (
	"go/ast"
	"go/types"
)

type TypeInfo struct {
	Package     string
	PackagePath string
	Name        string // in case of struct or func type declaration Name == Kind
	Kind        string // usually the way we Extract
	IsArray     bool   // if it's not array, it's struct
	IsPointer   bool   // if array, it's pointer
	IsImported  bool   // if kind it's an imported one
	IsAlias     bool   // if it's alias
	IsFunc      bool   // if it's a function type definition
	Fields      Fields
	MethodList  Methods
	comment     *ast.CommentGroup
}

func NewAliasFromField(pkg *types.Package, field *FieldInfo, name string) *TypeInfo {
	return &TypeInfo{
		Package:     pkg.Name(),
		PackagePath: pkg.Path(),
		Name:        name,
		Kind:        field.Kind,
		IsArray:     field.IsArray,
		IsPointer:   field.IsPointer,
		IsImported:  field.IsImported,
		IsAlias:     true,
	}
}

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
