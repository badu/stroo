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

type CodeConfig struct {
	SelectedType     string
	TestMode         bool
	DebugPrint       bool
	Serve            bool
	TemplateFile     string
	OutputFile       string
	SelectedPeerType string
}

type Command struct {
	CodeConfig
	CodeBuilder *analysis.Analyzer
	Inspector   *analysis.Analyzer
	WorkingDir  string
	Result      *PackageInfo
	Out         bytes.Buffer
}

// builds a new command from the analyzer (which holds the inspector) and sets the Run function
func NewCommand(analyzer *analysis.Analyzer) *Command {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("could NOT get working dir : %v", err)
	}
	if len(analyzer.Requires) != 1 {
		log.Fatalf("we require only inspectAlyzer - shouldn't happen")
	}
	result := Command{
		CodeConfig: CodeConfig{
			SelectedType:     analyzer.Flags.Lookup("type").Value.String(),
			TestMode:         analyzer.Flags.Lookup("testMode").Value.String() == "true",
			DebugPrint:       analyzer.Flags.Lookup("debugPrint").Value.String() == "true",
			Serve:            analyzer.Flags.Lookup("serve").Value.String() == "true",
			TemplateFile:     analyzer.Flags.Lookup("template").Value.String(),
			OutputFile:       analyzer.Flags.Lookup("output").Value.String(),
			SelectedPeerType: analyzer.Flags.Lookup("target").Value.String(),
		},
		WorkingDir:  workingDir,
		CodeBuilder: analyzer,
		Inspector:   analyzer.Requires[0], // needed in Run of the Command
	}
	analyzer.Run = result.Run // set the Run function to the analyzer
	return &result
}

func DefaultAnalyzer() *analysis.Analyzer {
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
	result := &analysis.Analyzer{
		Name:             toolName,
		Doc:              toolDoc,
		RunDespiteErrors: true,
		Requires:         []*analysis.Analyzer{inspectAlyzer},
		ResultType:       reflect.TypeOf(new(PackageInfo)),
	}
	result.Flags.Bool("serve", false, "serve the playground, to help you build templates")
	result.Flags.String("type", "", "type that should be processed e.g. SomeJsonPayload")
	result.Flags.String("output", "", "name of the output file e.g. json_gen.go")
	result.Flags.String("template", "", "name of the template file e.g. ./../templates/")
	result.Flags.String("target", "", "name of the peer struct e.g. ./../testdata/pkg/model_b/SomeProtoBufPayload")
	result.Flags.Bool("testMode", false, "is in test mode : just display the result")
	result.Flags.Bool("debugPrint", false, "print debugging info")
	result.Flags.Usage = func() {
		descMultiline := strings.Split(toolDoc, "\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "%s: %s\n\n", toolName, descMultiline[0])
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [-flag] [package]\n", toolName)
		if len(descMultiline) > 1 {
			_, _ = fmt.Fprintln(os.Stderr, strings.Join(descMultiline[1:], "\n\n"))
		}
		_, _ = fmt.Fprintf(os.Stderr, "\nFlags:\n")
		result.Flags.PrintDefaults()
	}
	return result
}

func Prepare() *Command {
	analyzer := DefaultAnalyzer()
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
func (c *Command) Print(withRunningFolder bool) string {
	result := ""
	if withRunningFolder {
		result = fmt.Sprintf("running in folder %q\n", c.WorkingDir)
	}
	c.CodeBuilder.Flags.VisitAll(func(f *flag.Flag) {
		if !withRunningFolder {
			result += "-"
		}
		result += f.Name + "=" + f.Value.String() + " "
	})
	return result
}

// check if vital things are missing from the configuration
func (c *Command) Check() {
	if c.TemplateFile == "" || c.SelectedType == "" || (!c.TestMode && c.OutputFile == "") {
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
	for _, imprt := range pass.Pkg.Imports() {
		result.Imports = append(result.Imports, imprt.Path())
	}
	result.TypesInfo = pass.TypesInfo // exposed just in case someone wants to get wild
	inspResult.Preorder(nodeFilter, func(node ast.Node) {
		if err != nil {
			return // we have error for a previous step
		}
		switch nodeType := node.(type) {
		case *ast.FuncDecl:
			if infoErr := result.ReadFunctionInfo(nodeType); infoErr != nil {
				err = infoErr
			}
		case *ast.GenDecl:
			switch nodeType.Tok {
			case token.TYPE:
				for _, spec := range nodeType.Specs {
					typeSpec := spec.(*ast.TypeSpec)
					switch unknownType := typeSpec.Type.(type) {
					case *ast.InterfaceType:
						// e.g. `type Intf interface{}`
						if infoErr := result.ReadInterfaceInfo(spec, nodeType.Doc); infoErr != nil {
							err = infoErr
						}
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
						if infoErr := result.DirectIdent(unknownType, nodeType.Doc); infoErr != nil {
							err = fmt.Errorf("error  : %v", infoErr)
						}
					case *ast.SelectorExpr:
						// e.g. : `type Timer time.Ticker`
						if infoErr := result.DirectSelector(unknownType, nodeType.Doc); infoErr != nil {
							err = fmt.Errorf("error : %v", infoErr)
						}
					case *ast.StarExpr:
						// e.g. : `type Timer *time.Ticker`
						if infoErr := result.DirectPointer(unknownType, nodeType.Doc); infoErr != nil {
							err = fmt.Errorf("error : %v", infoErr)
						}
					default:
						err = fmt.Errorf("have you modified the filter ? Unhandled : %#v\n", unknownType)
					}
				}
			case token.VAR, token.CONST:
				for _, spec := range nodeType.Specs {
					switch vl := spec.(type) {
					case *ast.ValueSpec:
						if infoErr := result.ReadVariablesInfo(spec, vl); infoErr != nil {
							err = infoErr
						}
					}
				}
			}
		}
	})
	return result, err
}

// load package
func (c *Command) Load(path string) (*packages.Package, error) {
	conf := packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Tests: true,
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
		return nil, fmt.Errorf("one pacakge at a time required")
	}
	return loadedPackage[0], nil
}

func (c *Command) Analyse(loadedPackage *packages.Package) error {
	type key struct {
		*analysis.Analyzer
		*packages.Package
	}

	actions := make(map[key]*Action)

	var mkAction func(analyzer *analysis.Analyzer, pkg *packages.Package) *Action
	mkAction = func(analyzer *analysis.Analyzer, pkg *packages.Package) *Action {
		mapKey := key{analyzer, pkg}
		action, ok := actions[mapKey]
		if !ok {
			action = &Action{parentAnalyzer: analyzer, pkg: pkg}
			for _, req := range analyzer.Requires {
				action.deps = append(action.deps, mkAction(req, pkg))
			}
			if len(analyzer.FactTypes) > 0 {
				paths := make([]string, 0, len(pkg.Imports))
				for path := range pkg.Imports {
					paths = append(paths, path)
				}
				sort.Strings(paths)
				for _, path := range paths {
					dep := mkAction(analyzer, pkg.Imports[path])
					action.deps = append(action.deps, dep)
				}
			}
			actions[mapKey] = action
		}
		return action
	}

	result := mkAction(c.CodeBuilder, loadedPackage)
	result.exec()

	if result.err != nil {
		return result.err
	}
	typedResult, ok := result.result.(*PackageInfo)
	if !ok {
		return fmt.Errorf("error : interface not *PackageInfo")
	}
	c.Result = typedResult
	return nil
}

func (c *Command) Generate() error {
	result, err := c.NewCode()
	if err != nil {
		return fmt.Errorf("error making code object : %v", err)
	}

	var buf bytes.Buffer
	if err := result.Tmpl().Execute(&buf, &result); err != nil {
		return fmt.Errorf("failed to parse template %s: %s\nPartial result:\n%s", c.TemplateFile, err, buf.String())
	}

	// forced add header
	var src []byte
	src = append(src, result.Header(c.Print(false))...)
	src = append(src, buf.Bytes()...)
	// format the source
	formatted, err := format.Source(src)
	if err != nil {
		return fmt.Errorf("go/format error: %v\nGo source:\n%s", err, src)
	}
	c.Out.Write(formatted)
	// if it's testmode, print and exit (same as playground, but in terminal)
	if c.TestMode {
		return nil
	}
	// TODO : if file exists, overwrite only the generated part - template should announce the intention of generator e.g. will write methods with signature "String() string" for the struct named "<struct_name>"
	/**
	if _, err := os.Stat(*OutputFile); !os.IsNotExist(err) {
		log.Fatalf("destination exists = %q", *OutputFile)
	}
	**/
	log.Printf("Creating %s/%s\n", c.WorkingDir, c.OutputFile)
	file, err := os.Create(c.WorkingDir + "/" + c.OutputFile)
	if err != nil {
		return fmt.Errorf("error creating file : %v", err)
	}
	// go ahead and write the file
	if _, err := file.Write(formatted); err != nil {
		return fmt.Errorf("error writing to file : %v", err)
	}
	return nil
}

func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
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
	}
}

func (c *Command) NewCode() (*Code, error) {
	templatePath, err := filepath.Abs(c.TemplateFile)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template-error : %v ; path = %q", err, templatePath)
	}
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(DefaultFuncMap()).ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("template-parse-error : %v ; path = %q", err, templatePath)
	}
	result := Code{
		PackageInfo: c.Result,
		CodeConfig:  c.CodeConfig,
		keeper:      make(map[string]interface{}),
		tmpl:        tmpl,
		Main:        TypeWithRoot{T: c.Result.Types.Extract(c.SelectedType)},
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

func (a *Action) ResultType() reflect.Type {
	return reflect.TypeOf(a.result)
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
			if got, want := a.ResultType(), a.pass.Analyzer.ResultType; got != want {
				err = fmt.Errorf("internal error: on package %s, analyzer %s returned a result of type %v, but declared ResultType %v", a.pass.Pkg.Path(), a.pass.Analyzer, got, want)
			}
		} else {
			log.Println("[error] : " + err.Error())
		}
	}
	a.err = err
}
