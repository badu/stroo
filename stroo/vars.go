package stroo

import "strings"

type VarInfo struct {
	Name string
	Type string
}

// cannot implement Stringer due to tests
func (v VarInfo) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	sb.WriteString("-----VAR-----\n")
	sb.WriteString(tabs + "Name:" + v.Name + "\n")
	sb.WriteString(tabs + "Type:" + v.Type + "\n")
}

type Vars []VarInfo

// cannot implement Stringer due to tests
func (v Vars) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	sb.WriteString(tabs + "Vars:Vars{\n")
	for _, vr := range v {
		if tno > 0 {
			vr.Debug(sb, tno)
		} else {
			vr.Debug(sb)
		}
	}
	sb.WriteString(tabs + "},\n")
}
