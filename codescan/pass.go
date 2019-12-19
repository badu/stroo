package codescan

// =====================================================================================================================
// Below code taken (and modified) from golang.org/x/tools
// =====================================================================================================================

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
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
