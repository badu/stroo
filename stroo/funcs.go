package stroo

import (
	"strings"
)

type FunctionInfo struct {
	Name         string
	ReturnType   string
	ReceiverName string
	ReceiverType string
	Package      string
	PackagePath  string
	IsExported   bool
}

// cannot implement Stringer due to tests
func (f FunctionInfo) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	sdb.WriteString(tabs + "&FunctionInfo{\n")
	sdb.WriteString(tabs + "Name:\"" + f.Name + "\",\n")
	if f.Package != "" {
		sdb.WriteString(tabs + "Package:\"" + f.Package + "\",\n")
	}
	if f.PackagePath != "" {
		sdb.WriteString(tabs + "PackagePath:\"" + f.PackagePath + "\",\n")
	}
	sdb.WriteString(tabs + "},\n")
}

type Methods []*FunctionInfo

// cannot implement Stringer due to tests
func (m Methods) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	if len(m) > 0 {
		sdb.WriteString(tabs + "&Methods{\n")
	}
	sdb.tabs++
	for _, meth := range m {
		meth.Debug(sdb)
	}
	sdb.tabs--
	if len(m) > 0 {
		sdb.WriteString(tabs + "},\n")
	}
}
