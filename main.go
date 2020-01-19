package stroo

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/badu/stroo/dbg_prn"
	"go/ast"
	"go/format"
	"go/token"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"text/template"
)

const (
	toolName = "stroo"
	toolDoc  = "extracts declaration of a struct with it's methods"
)

type Command struct {
	TypeName     string
	TestMode     bool
	DebugPrint   bool
	Serve        bool
	TemplateFile string
	OutputFile   string
	PeerStruct   string
	CodeBuilder  *analysis.Analyzer
	Inspector    *analysis.Analyzer
	Path         string
	WorkingDir   string
}

// builds a new command from the analyzer (which holds the inspector) and sets the Run function
func NewCommand(analyzer *analysis.Analyzer) *Command {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("could NOT get working dir : %v", err)
	}
	if len(analyzer.Requires) != 1 {
		panic("we require only inspectAlyzer - shouldn't happen")
	}
	result := Command{
		TypeName:     analyzer.Flags.Lookup("type").Value.String(),
		TestMode:     analyzer.Flags.Lookup("testMode").Value.String() == "true",
		DebugPrint:   analyzer.Flags.Lookup("debugPrint").Value.String() == "true",
		Serve:        analyzer.Flags.Lookup("serve").Value.String() == "true",
		TemplateFile: analyzer.Flags.Lookup("template").Value.String(),
		OutputFile:   analyzer.Flags.Lookup("output").Value.String(),
		PeerStruct:   analyzer.Flags.Lookup("target").Value.String(),
		WorkingDir:   workingDir,
		CodeBuilder:  analyzer,
		Inspector:    analyzer.Requires[0], // needed in Run of the Command
	}
	analyzer.Run = result.Run // set the Run function to the analyzer
	return &result
}

// set the flags to the analyzer
func setFlags(flagSet *flag.FlagSet) {
	flagSet.Bool("serve", false, "serve the playground, to help you build templates")
	flagSet.String("type", "", "type that should be processed e.g. SomeJsonPayload")
	flagSet.String("output", "", "name of the output file e.g. json_gen.go")
	flagSet.String("template", "", "name of the template file e.g. ./../templates/")
	flagSet.String("target", "", "name of the peer struct e.g. ./../testdata/pkg/model_b/SomeProtoBufPayload")
	flagSet.Bool("testMode", false, "is in test mode : just display the result")
	flagSet.Bool("debugPrint", false, "print debugging info")
	flagSet.Usage = func() {
		descMultiline := strings.Split(toolDoc, "\n\n")
		fmt.Fprintf(os.Stderr, "%s: %s\n\n", toolName, descMultiline[0])
		fmt.Fprintf(os.Stderr, "Usage: %s [-flag] [package]\n", toolName)
		if len(descMultiline) > 1 {
			fmt.Fprintln(os.Stderr, strings.Join(descMultiline[1:], "\n\n"))
		}
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flagSet.PrintDefaults()
	}
}

func Prepare() *Command {
	// traverse the ast tree providing result for our analyzer
	inspectAlyzer := &analysis.Analyzer{
		Name: "inspect",
		Doc:  "AST traversal for later passes",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return inspector.New(pass.Files), nil
		},
		RunDespiteErrors: true,
		ResultType:       reflect.TypeOf(new(inspector.Inspector)),
	}
	// the analyzer that loads code with data
	analyzer := &analysis.Analyzer{
		Name:             toolName,
		Doc:              toolDoc,
		RunDespiteErrors: true,
		Requires:         []*analysis.Analyzer{inspectAlyzer},
		ResultType:       reflect.TypeOf(new(PackageInfo)),
	}
	// set flags needed from command line
	setFlags(&analyzer.Flags)
	// set the logger
	log.SetFlags(0)
	log.SetPrefix(toolName + ": ")
	// check flags
	if err := analyzer.Flags.Parse(os.Args[1:]); err != nil {
		log.Fatalf("error parsing flags: %v", err)
	}
	// create a command from our analyzer (so we don't pass parameters around)
	return NewCommand(analyzer)
}

// print the current configuration
func (c *Command) Print() {
	log.Printf("running in folder %q\n", c.WorkingDir)
	log.Printf("processing type : %q - test mode : %t, printing debug : %t\n", c.TypeName, c.TestMode, c.DebugPrint)
	log.Printf("serve : %t; template %q ; output %q ; target %q\n", c.Serve, c.TemplateFile, c.OutputFile, c.PeerStruct)
}

// check if vital things are missing from the configuration
func (c *Command) Check() {
	if c.TemplateFile == "" || c.TypeName == "" || (!c.TestMode && c.OutputFile == "") {
		c.CodeBuilder.Flags.Usage()
		os.Exit(1)
	}
}

// analyzer will Run this and we're creating Package info
func (c *Command) Run(pass *analysis.Pass) (interface{}, error) {
	var err error
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
		(*ast.GenDecl)(nil),
	}
	result := &PackageInfo{
		Name:       pass.Pkg.Name(),
		PrintDebug: c.DebugPrint,
	}
	inspResult, ok := pass.ResultOf[c.Inspector].(*inspector.Inspector)
	if !ok {
		log.Fatalf("Inspector is not (*inspector.Inspector)")
	}
	result.TypesInfo = pass.TypesInfo // exposed just in case someone wants to get wild
	inspResult.Preorder(nodeFilter, func(node ast.Node) {
		if err != nil {
			return // we have error for a previous step
		}
		switch nodeType := node.(type) {
		case *ast.FuncDecl:
			result.ReadFunctionInfo(nodeType)
		case *ast.GenDecl:
			switch nodeType.Tok {
			case token.TYPE:
				for _, spec := range nodeType.Specs {
					typeSpec := spec.(*ast.TypeSpec)
					switch unknownType := typeSpec.Type.(type) {
					case *ast.InterfaceType:
						// e.g. `type Intf interface{}`
						result.ReadInterfaceInfo(spec, nodeType.Doc)
					case *ast.ArrayType:
						// e.g. `type Array []string`
						if infoErr := result.ReadArrayInfo(spec.(*ast.TypeSpec), nodeType.Doc); infoErr != nil {
							err = infoErr
						}
						// e.g. `type Stru struct {}`
					case *ast.StructType:
						if infoErr := result.ReadStructInfo(spec.(*ast.TypeSpec), nodeType.Doc); infoErr != nil {
							err = infoErr
						}
					case *ast.Ident:
						// e.g. : `type String string`
						result.DirectIdent(unknownType, nodeType.Doc)
					case *ast.SelectorExpr:
						// e.g. : `type Timer time.Ticker`
						result.DirectSelector(unknownType, nodeType.Doc)
					case *ast.StarExpr:
						// e.g. : `type Timer *time.Ticker`
						result.DirectPointer(unknownType, nodeType.Doc)
					default:
						log.Fatalf("Have you modified the filter ? Unhandled : %#v\n", unknownType)
					}
				}
			case token.VAR, token.CONST:
				for _, spec := range nodeType.Specs {
					switch vl := spec.(type) {
					case *ast.ValueSpec:
						result.ReadVariablesInfo(spec, vl)
					}
				}
			}
		}
	})
	return result, err
}

// load package and run generate
func (c *Command) Do() error {
	conf := packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Tests: true,
	}
	log.Printf("Loading %q\n", c.Path)
	loadedPackages, err := packages.Load(&conf, c.Path)
	if err != nil {
		log.Printf("error loading package %q : %v\n", c.Path, err)
		return err
	}

	if n := packages.PrintErrors(loadedPackages); n > 1 {
		err = fmt.Errorf("%d errors during loading", n)
	} else if n == 1 {
		err = fmt.Errorf("error during loading")
	} else if len(loadedPackages) == 0 {
		err = fmt.Errorf("%s matched no packages\n", c.Path)
	}

	type key struct {
		*analysis.Analyzer
		*packages.Package
	}

	actions := make(map[key]*Action)

	var mkAction func(analyzer *analysis.Analyzer, pkg *packages.Package) *Action
	mkAction = func(analyzer *analysis.Analyzer, pkg *packages.Package) *Action {
		k := key{analyzer, pkg}
		act, ok := actions[k]
		if !ok {
			act = &Action{parentAnalyzer: analyzer, pkg: pkg}
			for _, req := range analyzer.Requires {
				act.deps = append(act.deps, mkAction(req, pkg))
			}
			if len(analyzer.FactTypes) > 0 {
				paths := make([]string, 0, len(pkg.Imports))
				for path := range pkg.Imports {
					paths = append(paths, path)
				}
				sort.Strings(paths)
				for _, path := range paths {
					dep := mkAction(analyzer, pkg.Imports[path])
					act.deps = append(act.deps, dep)
				}
			}
			actions[k] = act
		}
		return act
	}

	var results []*Action

	for _, pkg := range loadedPackages {
		root := mkAction(c.CodeBuilder, pkg)
		results = append(results, root)
	}

	execAll(results)

	return c.Generate(results)
}

func (c *Command) Generate(results []*Action) error {
	if len(results) == 1 {
		packageInfo, ok := results[0].result.(*PackageInfo)
		if !ok {
			log.Fatalf("Error : interface not *PackageInfo")
		}
		wkdir := c.WorkingDir
		goPath := os.Getenv("GOPATH")
		goPathParts := strings.Split(goPath, ":")
		for _, part := range goPathParts {
			wkdir = strings.Replace(wkdir, part, "", -1)
		}
		if strings.HasPrefix(wkdir, "/src/") {
			wkdir = wkdir[5:]
		}

		result, err := c.NewCode(packageInfo)
		if err != nil {
			log.Fatalf("error making code object : %v", err)
		}

		var buf bytes.Buffer
		if err := result.Tmpl().Execute(&buf, &result); err != nil {
			log.Fatalf("failed to parse template %s: %s\nPartial result:\n%s", c.TemplateFile, err, buf.String())
		}
		// forced add header
		var src []byte
		src = append(src, result.Header()...)
		src = append(src, buf.Bytes()...)
		formatted, err := format.Source(src)
		if err != nil {
			log.Fatalf("go/format: %s\nResult:\n%s", err, src)
		} else if !c.TestMode {
			/**
			if _, err := os.Stat(*OutputFile); !os.IsNotExist(err) {
				log.Fatalf("destination exists = %q", *OutputFile)
			}
			**/
			log.Printf("Creating %s/%s\n", c.WorkingDir, c.OutputFile)
			file, err := os.Create(c.WorkingDir + "/" + c.OutputFile)
			if err != nil {
				log.Fatalf("Error creating output: %v", err)
			}
			// go ahead and write the file
			if _, err := file.Write(formatted); err != nil {
				log.Fatalf("error writing : %v", err)
			}
		} else {
			log.Println(string(formatted))
		}
	} else {
		log.Printf("%d actions. something went bad.", len(results))
	}
	return nil
}

func (c *Command) NewCode(packageInfo *PackageInfo) (*Code, error) {
	templatePath, err := filepath.Abs(c.TemplateFile)
	if err != nil {
		return nil, err
	}
	tmpl, err := loadTemplate(templatePath, template.FuncMap{
		//"toJsonName":    swag.ToJSONName, // TODO : import all, but make it field functionality
		"in":            contains,
		"empty":         empty,
		"nil":           isNil,
		"lowerInitial":  lowerInitial,
		"capitalize":    capitalize,
		"templateGoStr": templateGoStr,
		"contains":      strings.Contains,
		"trim":          strings.TrimSpace,
		"hasPrefix":     strings.HasPrefix,
		"sort":          SortFields, // allows fields sorting (tested in Stringer)
		"dump": func(a ...interface{}) string {
			return dbg_prn.SPrint(a...)
		},
		"concat": func(a, b string) string {
			return a + b
		},
	})
	if err != nil {
		return nil, err
	}
	result := Code{
		PackageInfo:  packageInfo,
		SelectedType: c.TypeName,
		OutputFile:   c.OutputFile,
		PeerName:     c.PeerStruct,
		TemplateFile: c.TemplateFile,
		TestMode:     c.TestMode,
		keeper:       make(map[string]interface{}),
		tmpl:         tmpl,
		debugPrint:   c.DebugPrint,
		Main:         TypeWithRoot{T: packageInfo.Types.Extract(c.TypeName)},
	}
	result.Main.D = &result
	return &result, nil
}

// below is copy paste (with some modifications) from golang.org/x/tools/go/analysis/internal/checker,
// because we cannot use that internal package
type Action struct {
	once           sync.Once
	parentAnalyzer *analysis.Analyzer
	pkg            *packages.Package
	pass           *analysis.Pass
	deps           []*Action
	inputs         map[*analysis.Analyzer]interface{}
	result         interface{}
	err            error
}

func (a *Action) String() string {
	return fmt.Sprintf("%s@%s", a.parentAnalyzer, a.pkg)
}

func execAll(actions []*Action) {
	var wg sync.WaitGroup
	for _, act := range actions {
		wg.Add(1)
		work := func(act *Action) {
			act.exec()
			wg.Done()
		}
		go work(act)
	}
	wg.Wait()
}

func (a *Action) exec() {
	a.once.Do(a.execOnce)
}

func (a *Action) execOnce() {
	execAll(a.deps)
	var failed []string
	for _, dep := range a.deps {
		if dep.err != nil {
			failed = append(failed, dep.String())
		}
	}
	if failed != nil {
		sort.Strings(failed)
		a.err = fmt.Errorf("failed pre-requisites: %s", strings.Join(failed, ", "))
		return
	}

	inputs := make(map[*analysis.Analyzer]interface{})

	for _, dep := range a.deps {
		if dep.pkg == a.pkg {
			inputs[dep.parentAnalyzer] = dep.result

		}
	}

	a.pass = &analysis.Pass{
		Analyzer:   a.parentAnalyzer,
		Fset:       a.pkg.Fset,
		Files:      a.pkg.Syntax,
		OtherFiles: a.pkg.OtherFiles,
		Pkg:        a.pkg.Types,
		TypesInfo:  a.pkg.TypesInfo,
		TypesSizes: a.pkg.TypesSizes,
		ResultOf:   inputs,
	}

	var err error
	if a.pkg.IllTyped && !a.pass.Analyzer.RunDespiteErrors {
		err = fmt.Errorf("analysis skipped due to errors in package")
	} else {
		a.result, err = a.pass.Analyzer.Run(a.pass)
		if err == nil {
			if got, want := reflect.TypeOf(a.result), a.pass.Analyzer.ResultType; got != want {
				err = fmt.Errorf("internal error: on package %s, analyzer %s returned a result of type %v, but declared ResultType %v", a.pass.Pkg.Path(), a.pass.Analyzer, got, want)
			}
		}
	}
	a.err = err
}
