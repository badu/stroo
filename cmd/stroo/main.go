package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/badu/stroo/codescan"

	. "github.com/badu/stroo/stroo"

	"github.com/go-openapi/swag"
)

var mAnalyzer = &codescan.Analyzer{
	Name: "inspect",
	Doc:  "AST traversal for later passes",
	Runnner: func(pass *codescan.Pass) (interface{}, error) {
		return codescan.NewInpector(pass.Files), nil
	},
	RunDespiteErrors: true,
	ResultType:       reflect.TypeOf(new(codescan.Inspector)),
}

func run(pass *codescan.Pass) (interface{}, error) {
	var (
		nodeFilter = []ast.Node{
			(*ast.FuncDecl)(nil),
			(*ast.FuncLit)(nil),
			(*ast.GenDecl)(nil),
		}
		err    error
		result = &PackageInfo{Name: pass.Pkg.Name(), StructDefs: make(map[string]*TypeInfo), FieldsDefs: make(map[string]*FieldInfo)}
	)

	inspector := pass.ResultOf[mAnalyzer].(*codescan.Inspector)
	defs := pass.TypesInfo.Defs
	inspector.Do(nodeFilter, func(node ast.Node) {
		if err != nil {
			return // we have error for a previous step
		}
		switch nodeType := node.(type) {
		case *ast.FuncDecl:
			result.ReadFunctionInfo(nodeType, defs[nodeType.Name])
		case *ast.GenDecl:
			switch nodeType.Tok {
			case token.TYPE:
				for _, spec := range nodeType.Specs {
					typeSpec := spec.(*ast.TypeSpec)
					switch unknownType := typeSpec.Type.(type) {
					case *ast.InterfaceType:
						result.ReadInterfaceInfo(spec, defs[typeSpec.Name], nodeType.Doc)
					case *ast.ArrayType:
						if infoErr := result.ReadArrayInfo(spec, defs[typeSpec.Name], nodeType.Doc); infoErr != nil {
							err = infoErr
						}
					case *ast.StructType:
						if infoErr := result.ReadStructInfo(spec, defs[typeSpec.Name], nodeType.Doc); infoErr != nil {
							err = infoErr
						}
					case *ast.Ident:
						info := FieldInfo{Name: defs[typeSpec.Name].Name(), Comment: nodeType.Doc}
						result.ReadIdent(unknownType, &info)
						result.FieldsDefs[info.Name] = &info
					default:
						log.Printf("Have you modified the filter ? Unhandled : %#v\n", unknownType)
					}
				}
			case token.VAR, token.CONST:
				for _, spec := range nodeType.Specs {
					obj := pass.TypesInfo.Defs[spec.(*ast.ValueSpec).Names[0]]
					switch vl := spec.(type) {
					case *ast.ValueSpec:
						result.ReadVariablesInfo(vl, obj)
					}
				}
			}
		}
	})

	if err != nil {
		return nil, err
	}
	return result, result.PostProcess()
}

var (
	typeName     = flag.String("type", "", "type that should be processed e.g. SomeJsonPayload")
	outputFile   = flag.String("output", "", "name of the output file e.g. json_gen.go")
	templateFile = flag.String("template", "", "name of the template file e.g. ./../templates/")
	peerStruct   = flag.String("target", "", "name of the peer struct e.g. ./../testdata/pkg/model_b/SomeProtoBufPayload")
	testMode     = flag.Bool("testmode", false, "is in test mode : just display the result") // todo make this true
)

func loadTemplate(path string, fnMap template.FuncMap) (*template.Template, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("error : %v ; path = %q", err, path)
	}
	return template.Must(template.New(filepath.Base(path)).Funcs(fnMap).ParseFiles(path)), nil
}

func contains(args ...string) bool {
	who := args[0]
	for i := 1; i < len(args); i++ {
		if args[i] == who {
			return true
		}
	}
	return false
}

func empty(src string) bool {
	if src == "" {
		return true
	}
	return false
}

func lowerInitial(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}

func templateGoStr(input string) string {
	if len(input) > 0 && input[len(input)-1] == '\n' {
		input = input[0 : len(input)-1]
	}
	if strings.Contains(input, "`") {
		lines := strings.Split(input, "\n")
		for idx, line := range lines {
			lines[idx] = strconv.Quote(line + "\n")
		}
		return strings.Join(lines, " + \n")
	}
	return "`" + input + "`"
}

type Doc struct {
	Imports          []string
	GeneratedMethods []string
	PackageInfo      *PackageInfo
	MainStruct       *TypeInfo
	RelatedStruct    *TypeInfo
	TypeName         string // from flags
	OutputFile       string // from flags
	TemplateFile     string // from flags
	PeerName         string // from flags
	TestMode         bool   // from flags
}

func (d *Doc) AddToImports(imp string) string {
	d.Imports = append(d.Imports, imp)
	return ""
}

func (d *Doc) AddToGeneratedMethods(methodName string) string {
	d.GeneratedMethods = append(d.GeneratedMethods, methodName)
	return ""
}

func (d *Doc) Header() string {
	flags := ""
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		flags += "-" + f.Name + "=" + f.Value.String() + " "
	})
	return fmt.Sprintf("// Generated on %v by Stroo [https://github.com/badu/stroo]\n"+
		"// Do NOT bother with altering it by hand : use the tool\n"+
		"// Arguments at the time of generation:\n//\t%s", time.Now().Format("Mon Jan 2 15:04:05"), flags)
}

func SortFields(fields Fields) bool {
	sort.Sort(fields)
	return true
}

func main() {
	analyzer := &codescan.Analyzer{
		Name:             "stroo",
		Doc:              "extracts declaration of a struct with it's methods",
		Flags:            flag.FlagSet{},
		Runnner:          run,
		RunDespiteErrors: true,
		Requires:         codescan.Analyzers{mAnalyzer},
		ResultType:       reflect.TypeOf(new(PackageInfo)),
	}

	log.SetFlags(0)
	log.SetPrefix(analyzer.Name + ": ")
	analyzers := codescan.Analyzers{analyzer}
	if err := analyzers.Validate(); err != nil {
		log.Fatal(err)
	}
	analyzers.ParseFlags()

	var (
		tmpl         *template.Template
		templatePath string
		err          error
	)
	templatePath, err = filepath.Abs(*templateFile)
	tmpl, err = loadTemplate(templatePath, template.FuncMap{
		"in":            contains,
		"empty":         empty,
		"lowerInitial":  lowerInitial,
		"capitalize":    capitalize,
		"templateGoStr": templateGoStr,
		"trim":          strings.TrimSpace,
		"toJsonName":    swag.ToJSONName, // TODO : import all, but make it field functionality
		"sort":          SortFields,      // TODO : test sort fields (fields implements the interface)
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Processing type : %q - test mode : %t\n", *typeName, *testMode)

	flag.Usage = func() {
		paras := strings.Split(analyzer.Doc, "\n\n")
		fmt.Fprintf(os.Stderr, "%s: %s\n\n", analyzer.Name, paras[0])
		fmt.Fprintf(os.Stderr, "Usage: %s [-flag] [package]\n\n", analyzer.Name)
		if len(paras) > 1 {
			fmt.Fprintln(os.Stderr, strings.Join(paras[1:], "\n\n"))
		}
		fmt.Fprintf(os.Stderr, "\nFlags:")
		flag.PrintDefaults()
	}

	args := flag.Args()
	if *templateFile == "" {
		log.Fatal("Error : you have to provide a template file (with relative path)")
	}
	if *typeName == "" {
		log.Fatal("Error : you have to provide a main type to be used in the template")
	}
	if !*testMode && *outputFile == "" {
		log.Fatal("Error : you have to specify the Go file which will be produced")
	}

	initial, err := codescan.Load(args)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	roots := initial.Analyze(analyzers)

	results, exitCode := roots.GatherResults()
	if len(results) == 1 {
		packageInfo, ok := results[0].(*PackageInfo)
		if !ok {
			log.Fatalf("Error : interface not *PackageInfo")
		}

		wkdir, _ := os.Getwd()
		originalWkDir := wkdir
		goPath := os.Getenv(codescan.GOPATH)
		goPathParts := strings.Split(goPath, ":")
		for _, part := range goPathParts {
			wkdir = strings.Replace(wkdir, part, "", -1)
		}
		if strings.HasPrefix(wkdir, codescan.SrcFPath) {
			wkdir = wkdir[5:]
		}
		doc := Doc{
			PackageInfo:  packageInfo,
			MainStruct:   packageInfo.GetStructByKey(*typeName),
			TypeName:     *typeName,
			OutputFile:   *outputFile,
			PeerName:     *peerStruct,
			TemplateFile: *templateFile,
			TestMode:     *testMode,
		}
		buf := bytes.Buffer{}
		if err := tmpl.Execute(&buf, &doc); err != nil {
			log.Fatalf("failed to parse template %s: %s", *templateFile, err)
		}
		// forced add header
		var src []byte
		src = append(src, doc.Header()...)
		src = append(src, buf.Bytes()...)
		formatted, err := format.Source(src)
		if err != nil {
			log.Fatalf("go/format: %s\nResult:\n%s", err, src)
		} else if !*testMode {
			/**
			if _, err := os.Stat(*outputFile); !os.IsNotExist(err) {
				log.Fatalf("destination exists = %q", *outputFile)
			}
			**/
			log.Printf("Creating %s/%s\n", originalWkDir, *outputFile)
			file, err := os.Create(originalWkDir + "/" + *outputFile)
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
	}
	os.Exit(exitCode)
}
