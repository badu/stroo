package stroo

import "strings"

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
func (f FunctionInfo) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	sb.WriteString(tabs + "&FunctionInfo{\n")
	sb.WriteString(tabs + "Name:\"" + f.Name + "\",\n")
	if f.Package != "" {
		sb.WriteString(tabs + "Package:\"" + f.Package + "\",\n")
	}
	if f.PackagePath != "" {
		sb.WriteString(tabs + "PackagePath:\"" + f.PackagePath + "\",\n")
	}
	sb.WriteString(tabs + "},\n")
}

type Methods []*FunctionInfo

// cannot implement Stringer due to tests
func (m Methods) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	if len(m) > 0 {
		sb.WriteString(tabs + "&Methods{\n")
	}
	for _, meth := range m {
		if tno > 0 {
			meth.Debug(sb, tno)
		} else {
			meth.Debug(sb)
		}
	}
	if len(m) > 0 {
		sb.WriteString(tabs + "},\n")
	}
}
