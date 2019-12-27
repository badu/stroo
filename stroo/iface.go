package stroo

import (
	"go/ast"
	"strings"
)

type InterfaceInfo struct {
	Name        string
	Methods     []string
	Package     string
	PackagePath string
	Comment     *ast.CommentGroup
}

// cannot implement Stringer due to tests
func (i InterfaceInfo) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	sb.WriteString("-----INTERFACE-----\n")
	sb.WriteString(tabs + "Name:" + i.Name + "\n")
	for _, meth := range i.Methods {
		sb.WriteString(tabs + "Method:" + meth + "\n")
	}
}

type Interfaces []*InterfaceInfo

// cannot implement Stringer due to tests
func (i Interfaces) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	if len(i) > 0 {
		sb.WriteString(tabs + "&Interfaces{\n")
	}
	for _, ifa := range i {
		if tno > 0 {
			ifa.Debug(sb, tno)
		} else {
			ifa.Debug(sb)
		}
	}
	if len(i) > 0 {
		sb.WriteString(tabs + "},\n")
	}
}
