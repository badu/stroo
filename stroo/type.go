package stroo

import (
	"go/ast"
	"strings"
)

type TypeInfo struct {
	Comment            *ast.CommentGroup
	Package            string
	PackagePath        string
	Name               string
	TypeName           string
	Fields             Fields
	MethodList         Methods
	Reference          *TypeInfo
	ReferenceIsPointer bool
}

// cannot implement Stringer due to tests
func (s TypeInfo) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	sb.WriteString(tabs + "&TypeInfo{\n")
	sb.WriteString(tabs + "Name:\"" + s.Name + "\",\n")
	if s.Package != "" {
		sb.WriteString(tabs + "Package:\"" + s.Package + "\",\n")
	}
	if s.PackagePath != "" {
		sb.WriteString(tabs + "PackagePath:\"" + s.PackagePath + "\",\n")
	}
	if s.ReferenceIsPointer {
		sb.WriteString(tabs + "ReferenceIsPointer:true,\n")
	}
	if tno > 0 {
		s.Fields.Debug(sb, tno)
	} else {
		s.Fields.Debug(sb)
	}
	sb.WriteString(tabs + "},\n")
}

type TypesSlice []*TypeInfo

// cannot implement Stringer due to tests
func (s TypesSlice) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	sb.WriteString(tabs + "TypesSlice:TypesSlice{\n")
	for _, el := range s {
		if tno > 0 {
			el.Debug(sb, tno)
		} else {
			el.Debug(sb)
		}
	}
	sb.WriteString(tabs + "},\n")
}

func (s TypesSlice) Extract(typeName string) *TypeInfo {
	for _, typeDef := range s {
		if typeDef.Name == typeName {
			return typeDef
		}
	}
	return nil
}
