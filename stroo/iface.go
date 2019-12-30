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
func (i InterfaceInfo) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	sdb.WriteString("-----INTERFACE-----\n")
	sdb.WriteString(tabs + "Name:" + i.Name + "\n")
	for _, meth := range i.Methods {
		sdb.WriteString(tabs + "Method:" + meth + "\n")
	}
}

type Interfaces []*InterfaceInfo

// cannot implement Stringer due to tests
func (i Interfaces) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	if len(i) > 0 {
		sdb.WriteString(tabs + "&Interfaces{\n")
	}
	sdb.tabs++
	for _, ifa := range i {
		ifa.Debug(sdb)
	}
	sdb.tabs--
	if len(i) > 0 {
		sdb.WriteString(tabs + "},\n")
	}
}
