package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/badu/stroo/dbg_prn"
	"go/ast"
	"go/format"
	"go/token"
	"io/ioutil"
	"log"
	"net/http"
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

	"github.com/fsnotify/fsnotify"
	"github.com/knadh/stuffbin"
	"github.com/r3labs/sse"
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
		result = &PackageInfo{
			Name:       pass.Pkg.Name(),
			PrintDebug: mAnalyzer.PrintDebug,
		}
	)
	inspector, ok := pass.ResultOf[mAnalyzer].(*codescan.Inspector)
	if !ok {
		log.Fatalf("Inspector is not (*codescan.Inspector)")
	}
	result.TypesInfo = pass.TypesInfo // exposed just in case someone wants to get wild
	//	var sb strings.Builder
	//	pass.Debug(&sb)
	//	log.Println(sb.String())
	inspector.Do(nodeFilter, func(node ast.Node) {
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
						log.Printf("Have you modified the filter ? Unhandled : %#v\n", unknownType)
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

func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	return false
}

type TypeWithRoot struct {
	T *TypeInfo // like "current" type
	D *Doc      // required as "parent" in recursion templates
}

type Doc struct {
	Imports      []string
	PackageInfo  *PackageInfo
	Main         TypeWithRoot
	SelectedType string                 // from flags
	OutputFile   string                 // from flags
	TemplateFile string                 // from flags
	PeerName     string                 // from flags
	TestMode     bool                   // from flags
	keeper       map[string]interface{} // template authors keeps data in here, key-value, as they need
	tmpl         *template.Template     // reference to template, so we don't pass it as parameter
	templateName string                 // set by template, used in GenerateAndStore and ListStored
}

func (d *Doc) StructByKey(key string) *TypeInfo {
	return d.PackageInfo.Types.Extract(key)
}

// returns true if the key exist and will overwrite
func (d *Doc) Store(key string, value interface{}) bool {
	_, has := d.keeper[key]
	d.keeper[key] = value
	return has
}

func (d *Doc) Retrieve(key string) interface{} {
	value, _ := d.keeper[key]
	return value
}

func (d *Doc) HasInStore(key string) bool {
	_, has := d.keeper[key]
	if *debugPrint {
		log.Printf("Has in store %q = %t", key, has)
	}
	return has
}

func (d *Doc) Keeper() map[string]interface{} {
	return d.keeper
}

func (d *Doc) AddToImports(imp string) string {
	d.Imports = append(d.Imports, imp)
	return ""
}

func (d *Doc) Declare(name string) bool {
	if d.Main.T == nil {
		log.Fatalf("error : main type is not set - impossible...")
	}
	if name == "" {
		log.Fatalf("error : cannot declare empty template name (e.g.`Stringer`)")
	}
	d.templateName = name
	d.keeper[name+d.Main.T.Name] = "" // set it to empty in case of self reference, so template will exit
	return true
}

func (d *Doc) GenerateAndStore(kind string) bool {
	entity := d.templateName + kind
	if *debugPrint {
		log.Printf("Processing %q %q ", d.templateName, kind)
	}
	// already has it
	if _, has := d.keeper[entity]; has {
		if *debugPrint {
			log.Printf("%q already stored.", kind)
		}
		return false
	}
	var buf strings.Builder
	nt := d.PackageInfo.Types.Extract(kind)
	if nt == nil {
		if *debugPrint {
			log.Printf("%q doesn't exist.", kind)
		}
		return false
	}

	err := d.tmpl.ExecuteTemplate(&buf, d.templateName, TypeWithRoot{D: d, T: nt})
	if err != nil {
		if *debugPrint {
			log.Printf("generate and store error : %v", err)
		}
		return false
	}
	d.keeper[entity] = buf.String()
	if *debugPrint {
		log.Printf("%q stored.", kind)
	}
	return true
}

func (d *Doc) ListStored() []string {
	var result []string
	for key, value := range d.keeper {
		if strings.HasPrefix(key, d.templateName) {
			if r, ok := value.(string); ok {
				// len(0) is default template for main (
				if len(r) > 0 {
					result = append(result, r)
				}
			} else {
				// if it's not a string, we're ignoring it
				if *debugPrint {
					log.Printf("%q has prefix %q but it's not a string and we're ignoring it", key, d.templateName)
				}
			}
		}
	}
	return result
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

var (
	typeName     = flag.String("type", "", "type that should be processed e.g. SomeJsonPayload")
	outputFile   = flag.String("output", "", "name of the output file e.g. json_gen.go")
	templateFile = flag.String("template", "", "name of the template file e.g. ./../templates/")
	peerStruct   = flag.String("target", "", "name of the peer struct e.g. ./../testdata/pkg/model_b/SomeProtoBufPayload")
	testMode     = flag.Bool("testmode", false, "is in test mode : just display the result")
	debugPrint   = flag.Bool("debug", false, "print debugging info")
	serve        = flag.Bool("serve", false, "serve the debug version")
)

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
	if *serve {
		var initFileWatcher = func(paths []string) (*fsnotify.Watcher, error) {
			fw, err := fsnotify.NewWatcher()
			if err != nil {
				return fw, err
			}

			files := []string{}
			// Get all the files which needs to be watched.
			for _, p := range paths {
				m, err := filepath.Glob(p)
				if err != nil {
					return fw, err
				}
				files = append(files, m...)
			}

			// Watch all template files.
			for _, f := range files {
				if err := fw.Add(f); err != nil {
					return fw, err
				}
			}
			return fw, err
		}
		// Initialize file watcher.
		var paths []string
		fw, err := initFileWatcher(paths)
		defer fw.Close()
		if err != nil {
			log.Printf("error watching files: %v", err)
			os.Exit(1)
		}

		var initSSEServer = func() *sse.Server {
			server := sse.New()
			server.CreateStream("messages")
			go func() {
				for {
					select {
					// Watch for events.
					case _ = <-fw.Events:
						log.Printf("files changed")
						// Send a ping notify frontent about file changes.
						server.Publish("messages", &sse.Event{
							Data: []byte("-"),
						})
						// Watch for errors.
					case err := <-fw.Errors:
						log.Printf("error watching files: %v", err)
					}
				}
			}()
			return server
		}

		// Initialize SSE server.
		server := initSSEServer()

		var initFileSystem = func() (stuffbin.FileSystem, error) {
			wd, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			// Read stuffed data from self.
			fs, err := stuffbin.UnStuff(os.Args[0])
			if err != nil {
				if err == stuffbin.ErrNoID {
					fs, err = stuffbin.NewLocalFS(wd, "./../../serve/index.html:index.html")
					if err != nil {
						log.Printf("Working folder : %q\nerror:%v", wd, err)
						return fs, fmt.Errorf("error falling back to local filesystem: %v", err)
					}
				} else {
					return fs, fmt.Errorf("error reading stuffed binary: %v", err)
				}
			}
			return fs, nil
		}

		var decodeTemplateData = func(dataRaw []byte) (map[string]interface{}, error) {
			var data map[string]interface{}
			err := json.Unmarshal(dataRaw, &data)
			if err != nil {
				return data, err
			}
			return data, nil
		}

		type Resp struct {
			Data  json.RawMessage `json:"data,omitempty"`
			Error string          `json:"error,omitempty"`
		}

		var writeJSONResp = func(w http.ResponseWriter, statusCode int, data []byte, e string) {
			var resp = Resp{
				Data:  json.RawMessage(data),
				Error: e,
			}
			b, err := json.Marshal(resp)
			if err != nil {
				log.Printf("error encoding response: %v, data: %v", err, string(data))
			}
			w.WriteHeader(statusCode)
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		}

		var tmplData map[string]interface{}
		var handleGetTemplateData = func(w http.ResponseWriter, r *http.Request) {
			d, err := json.Marshal(tmplData)
			if err != nil {
				writeJSONResp(w, http.StatusInternalServerError, nil, fmt.Sprintf("Error reading request body: %v", err))
			} else {
				writeJSONResp(w, http.StatusOK, d, "")
			}
		}

		var handleUpdateTemplateData = func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				writeJSONResp(w, http.StatusBadRequest, nil, fmt.Sprintf("Error reading request body: %v", err))
				return
			}

			data, err := decodeTemplateData(body)
			if err != nil {
				writeJSONResp(w, http.StatusBadRequest, nil, fmt.Sprintf("Error parsing JSON data: %v", err))
				return
			}

			// Update template data.
			//tmplDataMux.Lock()
			tmplData = data
			//tmplDataMux.Unlock()

			// Publish a message to reload.
			server.Publish("messages", &sse.Event{
				Data: []byte("-"),
			})

			writeJSONResp(w, http.StatusOK, nil, "")
		}

		var handleGetTemplateFields = func(w http.ResponseWriter, r *http.Request) {
			/**
			t, err := GetTemplate(tmplPath, baseTmplPaths)
			if err != nil {
				writeJSONResp(w, http.StatusInternalServerError, nil, fmt.Sprintf("Error getting template fields: %v", err))
				return
			}
			fields := NodeFields(t)
			mFields := []string{}
			for _, f := range fields {
				if nodeFieldRe.MatchString(f) {
					mFields = append(mFields, f)
				}
			}
			**/
			mFields := []string{}
			b, err := json.Marshal(mFields)
			if err != nil {
				writeJSONResp(w, http.StatusInternalServerError, nil, fmt.Sprintf("Error encoding template fields: %v", err))
				return
			}
			writeJSONResp(w, http.StatusOK, b, "")
		}

		mux := http.NewServeMux()
		// Initialize file system.
		fs, err := initFileSystem()
		if err != nil {
			log.Printf("error initializing file system: %v", err)
			os.Exit(1)
		}
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			f, err := fs.Get("index.html")
			if err != nil {
				log.Fatalf("error reading foo.txt: %v", err)
			}
			// Write response.
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html")
			w.Write(f.ReadBytes())
		})

		mux.HandleFunc("/events", server.HTTPHandler)

		// Handler to render template.
		mux.HandleFunc("/out", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				// Write response.
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			// Compile the template.
			var err error
			err = errors.New("not implemented")
			var b []byte
			//b, err := gotp.Compile(tmplPath, baseTmplPaths, tmplData)
			// If error send error as output.
			if err != nil {
				b = []byte(fmt.Sprintf("error rendering: %v", err))
			}
			// Write response.
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html")
			w.Write(b)
		})

		mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				handleUpdateTemplateData(w, r)
				return
			} else if r.Method == http.MethodGet {
				handleGetTemplateData(w, r)
			} else {
				// Write response.
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		})

		mux.HandleFunc("/fields", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				handleGetTemplateFields(w, r)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		})

		// Start server on given port.
		if err := http.ListenAndServe("localhost:8080", mux); err != nil {
			panic(err)
		}
	}
	var (
		tmpl         *template.Template
		templatePath string
		err          error
	)
	templatePath, err = filepath.Abs(*templateFile)

	log.Printf("Processing type : %q - test mode : %t, printing debug : %t\n", *typeName, *testMode, *debugPrint)

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
	tmpl, err = loadTemplate(templatePath, template.FuncMap{
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
		log.Fatal(err)
	}
	if *typeName == "" {
		log.Fatal("Error : you have to provide a main type to be used in the template")
	}
	if !*testMode && *outputFile == "" {
		log.Fatal("Error : you have to specify the Go file which will be produced")
	}
	mAnalyzer.PrintDebug = *debugPrint

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
			SelectedType: *typeName,
			OutputFile:   *outputFile,
			PeerName:     *peerStruct,
			TemplateFile: *templateFile,
			TestMode:     *testMode,
			keeper:       make(map[string]interface{}),
			tmpl:         tmpl,
			Main:         TypeWithRoot{T: packageInfo.Types.Extract(*typeName)},
		}

		doc.Main.D = &doc

		buf := bytes.Buffer{}
		if err := tmpl.Execute(&buf, &doc); err != nil {
			log.Fatalf("failed to parse template %s: %s\nPartial result:\n%s", *templateFile, err, buf.String())
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
