package stroo

import (
	"flag"
	"fmt"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
	"log"
	"os"
)

// loads one package
func LoadPackage(path string) (*packages.Package, error) {
	conf := packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Tests: false, //TODO : decide if tests are useful
	}

	loadedPackage, err := packages.Load(&conf, path) //supports variadic multiple paths but we're using only one
	if err != nil {
		log.Printf("error loading package %q : %v\n", path, err)
		return nil, err
	}
	allErrors := ""
	n := 0
	packages.Visit(loadedPackage, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			allErrors += err.Error() + "\n"
			n++
		}
	})

	switch n {
	case 0:
	default:
		return nil, fmt.Errorf("%d error(s) encountered during load:\n%s", n, allErrors)
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

func IsBasic(kind string) bool {
	for _, basic := range []string{
		"unsafe.Pointer", "bool", "byte",
		"complex64", "complex128",
		"error",
		"float32", "float64",
		"int", "int8", "int16", "int32", "int64",
		"rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr"} {
		if basic == kind {
			return true
		}
	}
	return false
}

func IsComplex(kind string) bool {
	for _, basic := range []string{"complex64", "complex128"} {
		if basic == kind {
			return true
		}
	}
	return false
}

func IsFloat(kind string) bool {
	for _, basic := range []string{"float32", "float64"} {
		if basic == kind {
			return true
		}
	}
	return false
}

func IsInt(kind string) bool {
	for _, basic := range []string{"int", "int8", "int16", "int32", "int64"} {
		if basic == kind {
			return true
		}
	}
	return false
}

func IsUint(kind string) bool {
	for _, basic := range []string{"uint", "uint8", "uint16", "uint32", "uint64", "uintptr"} {
		if basic == kind {
			return true
		}
	}
	return false
}
