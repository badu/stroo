package stroo

import (
	"go/ast"
	"strings"
)

type TypeInfo struct {
	Comment     *ast.CommentGroup
	Package     string
	PackagePath string
	Name        string
	TypeName    string
	Fields      Fields
	MethodList  Methods
}

// cannot implement Stringer due to tests
func (s TypeInfo) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
	}
	sb.WriteString(tabs + "&TypeInfo{\n")
	sb.WriteString(tabs + "Name:\"" + s.Name + "\",\n")
	if s.Package != "" {
		sb.WriteString(tabs + "Package:\"" + s.Package + "\",\n")
	}
	if s.PackagePath != "" {
		sb.WriteString(tabs + "PackagePath:\"" + s.PackagePath + "\",\n")
	}
	if len(s.Fields) > 0 {
		tno++
		if tno > 0 {
			s.Fields.Debug(sb, tno)
		} else {
			s.Fields.Debug(sb)
		}
	}
	sb.WriteString("},\n")
}

type TypesSlice []*TypeInfo

// cannot implement Stringer due to tests
func (s TypesSlice) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		if len(s) > 0 {
			tno++
		}
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
