package stroo

import (
	"flag"
	"fmt"
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
	"log"
	"os"
)

func getReceiver(fnDecl *ast.FuncDecl) interface{} {
	if fnDecl.Recv == nil {
		return nil
	}
	if len(fnDecl.Recv.List[0].Names) == 0 {
		return nil
	}
	return fnDecl.Recv.List[0].Names[0].Name
}

func getReceiverType(f *ast.FuncDecl) interface{} {
	if f.Recv == nil {
		return nil
	}
	if len(f.Recv.List[0].Names) == 0 {
		return nil
	}
	switch f.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return f.Recv.List[0].Type.(*ast.Ident).Name
	case *ast.StarExpr:
		return f.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
	}
	return nil
}

func stringer(str interface{}) string {
	if str == nil {
		return ""
	}
	return str.(string)
}

// loads one package
func LoadPackage(path string) (*packages.Package, error) {
	conf := packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Tests: false, //TODO : decide if tests are usefull
	}

	// did you knew that you can `loadedPackages, err := packages.Load(config, fmt.Sprintf("file=%s", filename))`
	loadedPackage, err := packages.Load(&conf, path) //supports variadic multiple paths but we're using only one
	if err != nil {
		log.Printf("error loading package %q : %v\n", path, err)
		return nil, err
	}

	n := packages.PrintErrors(loadedPackage)
	switch n {
	case 0:
	default:
		return nil, fmt.Errorf("%d error(s) encountered during load", n)
	}

	switch len(loadedPackage) {
	case 0:
		return nil, fmt.Errorf("%q matched no packages\n", path)
	case 1:
		// only allowed one
	default:
		for _, p := range loadedPackage {
			log.Printf("Package: %q\n", p.ID)
		}
		return nil, fmt.Errorf("%d packages found. we're looking for exactly one", len(loadedPackage))
	}
	return loadedPackage[0], nil
}

// print the current configuration
func Print(analyzer *analysis.Analyzer, withRunningFolder bool) string {
	result := ""
	if withRunningFolder {
		workingDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("could NOT get working dir : %v", err)
		}
		result = fmt.Sprintf("running in folder %q\n", workingDir)
	}
	analyzer.Flags.VisitAll(func(f *flag.Flag) {
		if !withRunningFolder {
			result += "-"
		}
		result += f.Name + "=" + f.Value.String() + " "
	})
	return result
}
