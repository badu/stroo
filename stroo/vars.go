package stroo

import "strings"

type VarInfo struct {
	Name string
	Type string
}

// cannot implement Stringer due to tests
func (v VarInfo) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	sdb.WriteString("-----VAR-----\n")
	sdb.WriteString(tabs + "Name:" + v.Name + "\n")
	sdb.WriteString(tabs + "Type:" + v.Type + "\n")
}

type Vars []VarInfo

// cannot implement Stringer due to tests
func (v Vars) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	sdb.WriteString(tabs + "Vars:Vars{\n")
	sdb.tabs++
	for _, vr := range v {
		vr.Debug(sdb)
	}
	sdb.tabs--
	sdb.WriteString(tabs + "},\n")
}
