package codescan

// =====================================================================================================================
// Below code taken (and modified) from golang.org/x/tools
// =====================================================================================================================

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

type Pass struct {
	Pkg        *types.Package
	SourceFile *token.FileSet
	Files      []*ast.File
	Analyzer   *Analyzer
	TypesInfo  *types.Info
	TypesSizes types.Sizes
	ResultOf   map[*Analyzer]interface{}
}

func (p *Pass) String() string {
	return fmt.Sprintf("%s@%s", p.Analyzer.Name, p.Pkg.Path())
}

func (p *Pass) Debug(sb *strings.Builder) {
	sb.WriteString("\n")
	for key, value := range p.TypesInfo.Defs {
		if value == nil {
			sb.WriteString(fmt.Sprintf("%q NIL VALUE\n", key.Name))
			continue
		}
		if value.Exported() {
			sb.WriteString(fmt.Sprintf("Key : %#v\n", key))
			sb.WriteString(fmt.Sprintf("\t%q exported id = %q\n", value.Name(), value.Id()))
			if value.Type() != nil {
				sb.WriteString(fmt.Sprintf("\tunderlying : %#v\n", value.Type().Underlying()))
			}
			if value.Pkg() != nil {
				sb.WriteString(fmt.Sprintf("\tpackage : %#v\n", value.Pkg()))
			}
		}
	}
}
