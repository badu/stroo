package stroo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/badu/stroo/statik"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"go/format"
	"go/parser"
	"go/token"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
)

const (
	playgroundVersion = "0.0.2"

	storageFolder = "/server"
	assetsFolder  = "/assets/"

	indexHTML        = "/index.html"
	codeTextAreaHTML = "/codetextarea.html"
	playgroundHTML   = "/playground.html"
	favico           = "/favicon.ico"
	exampleGo        = "/example-source"
	exampleTemplate  = "/example-template"

	jQuery          = "/jquery-3.4.1.js"
	semanticJs      = "/semantic-2.4.2.js"
	codeMirrorJs    = "/codemirror-5.51.0.js"
	riotJs          = "/riotcompiler-4.8.7.js"
	matchBracketsJs = "/matchbrackets.js"
	goJs            = "/go.js"

	semanticCss     = "/semantic-2.4.2.css"
	codeMirrorTheme = "/darcula-5.51.0.css"
	codeMirrorCss   = "/codemirror-5.51.0.css"

	font1 = "/themes/default/assets/fonts/icons.woff2"
	font2 = "/themes/default/assets/fonts/icons.woff"
	font3 = "/themes/default/assets/fonts/icons.ttf"

	packageName = "playground"
)

func provideFile(w http.ResponseWriter, statikFS http.FileSystem, path string) {
	r, err := statikFS.Open(path)
	if err != nil {
		log.Printf("error finding file : %q", path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer r.Close()
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		log.Printf("error reading file : %q", path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(contents)
}

func indexTemplate(statikFS http.FileSystem) *template.Template {
	r, err := statikFS.Open(indexHTML)
	if err != nil {
		log.Fatalf("error finding file : %q", indexHTML)
	}
	defer r.Close()
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("error reading file : %q", indexHTML)
	}
	index, err := template.New("index.html").Parse(string(contents))
	if err != nil {
		log.Fatalf("error parsing template : %q", indexHTML)
	}
	return index
}

func indexTemplateLocal(workingDir string) *template.Template {
	index, err := template.ParseFiles(workingDir + storageFolder + indexHTML)
	if err != nil {
		log.Fatalf("Error parsing template : %v\n working dir is = %q", err, workingDir)
	}
	return index
}

func filesHandler(workingDir string, statikFS http.FileSystem, testMode bool) http.HandlerFunc {
	type pageinfo struct {
		Version string
	}
	info := pageinfo{
		Version: playgroundVersion,
	}
	var idxTemplate *template.Template
	// prepare index.html
	if !testMode {
		idxTemplate = indexTemplate(statikFS)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case indexHTML, "/":
			if testMode {
				idxTemplate = indexTemplateLocal(workingDir) // reload template
			}
			if err := idxTemplate.ExecuteTemplate(w, "index", info); err != nil {
				log.Printf("error producing index template : %v", err)
				return
			}
		case codeTextAreaHTML, playgroundHTML:
			if testMode {
				http.ServeFile(w, r, workingDir+storageFolder+"/"+r.URL.Path)
			} else {
				provideFile(w, statikFS, r.URL.Path)
			}
		case semanticCss, codeMirrorCss, codeMirrorTheme:
			w.Header().Set("Content-Type", "text/css")
			provideFile(w, statikFS, r.URL.Path)
		case jQuery, semanticJs, codeMirrorJs, riotJs, goJs, matchBracketsJs:
			provideFile(w, statikFS, r.URL.Path)
		case font1, font2, font3:
			provideFile(w, statikFS, assetsFolder+strings.Replace(r.URL.Path, "/themes/default/assets/fonts/", "", -1))
		case exampleGo:
			if testMode {
				http.ServeFile(w, r, workingDir+storageFolder+exampleGo+".go")
			} else {
				provideFile(w, statikFS, exampleGo+".go")
			}
		case exampleTemplate:
			if testMode {
				http.ServeFile(w, r, workingDir+storageFolder+exampleTemplate+".tpl")
			} else {
				provideFile(w, statikFS, exampleTemplate+".tpl")
			}
		case favico:
		// just ignore it
		default:
			w.WriteHeader(http.StatusNotFound)
			log.Printf("Requested UNKNOWN URL : %q", r.URL.Path)
		}
	})
}

type ErrorType int

const (
	Json           ErrorType = 1
	TemplaParse    ErrorType = 2
	BadTempProject ErrorType = 3
	PackaLoad      ErrorType = 4
	OnePackage     ErrorType = 5
	Packalyse      ErrorType = 6
	NoTypes        ErrorType = 7
	TemplExe       ErrorType = 8
	BadFormat      ErrorType = 9
)

type MalformedRequest struct {
	Status        int       `json:"status"`
	ErrorMessage  string    `json:"errorMessage"`
	PartialSource string    `json:"partialSource"`
	Type          ErrorType `json:"type"`
}

func (m MalformedRequest) Error() string {
	return m.ErrorMessage
}

var InvalidGoSource = MalformedRequest{Type: BadTempProject, Status: http.StatusBadRequest}
var InvalidTypes = MalformedRequest{Type: NoTypes, Status: http.StatusBadRequest}
var InvalidPackage = MalformedRequest{Type: PackaLoad, Status: http.StatusBadRequest}
var InvalidAnalysis = MalformedRequest{Type: Packalyse, Status: http.StatusBadRequest}
var InvalidFormat = MalformedRequest{Type: BadFormat, Status: http.StatusBadRequest}
var InvalidTemplate2 = MalformedRequest{Type: TemplaParse, Status: http.StatusBadRequest}
var InvalidTemplate = MalformedRequest{Type: TemplExe, Status: http.StatusBadRequest}

func respond(w http.ResponseWriter, data interface{}, optionalMessage ...string) {
	if err, ok := data.(error); ok {
		var (
			typedError         MalformedRequest
			syntaxError        *json.SyntaxError
			unmarshalTypeError *json.UnmarshalTypeError
		)
		switch {
		case errors.Is(err, io.EOF):
			typedError = MalformedRequest{
				ErrorMessage: "payload empty",
				Status:       http.StatusBadRequest,
				Type:         Json,
			}
		case errors.As(err, &syntaxError):
		case errors.As(err, &unmarshalTypeError):
			typedError = MalformedRequest{
				ErrorMessage: err.Error(),
				Status:       http.StatusBadRequest,
				Type:         Json,
			}
		case errors.Is(err, InvalidGoSource):
			typedError = InvalidGoSource
			if len(optionalMessage) == 1 {
				typedError.ErrorMessage = optionalMessage[0]
			}
		case errors.Is(err, InvalidTypes):
			typedError = InvalidTypes
			if len(optionalMessage) == 1 {
				typedError.ErrorMessage = optionalMessage[0]
			}
		case errors.Is(err, InvalidAnalysis):
			typedError = InvalidAnalysis
			if len(optionalMessage) == 1 {
				typedError.ErrorMessage = optionalMessage[0]
			}
		case errors.Is(err, InvalidFormat):
			typedError = InvalidFormat
			switch len(optionalMessage) {
			case 1:
				typedError.ErrorMessage = optionalMessage[0]
			case 2:
				typedError.ErrorMessage = optionalMessage[0]
				typedError.PartialSource = optionalMessage[1]
			}
		case errors.Is(err, InvalidTemplate):
			typedError = InvalidTemplate
			if len(optionalMessage) == 1 {
				typedError.ErrorMessage = optionalMessage[0]
			}
		case errors.Is(err, InvalidTemplate2):
			typedError = InvalidTemplate2
			if len(optionalMessage) == 1 {
				typedError.ErrorMessage = optionalMessage[0]
			}
		case errors.Is(err, InvalidPackage):
			typedError = InvalidPackage
			if len(optionalMessage) == 1 {
				typedError.ErrorMessage = optionalMessage[0]
			}
		default:
			log.Printf("Unhandled error ? %T %#v", data, data)
			typedError = MalformedRequest{
				ErrorMessage: err.Error(),
				Status:       http.StatusBadRequest,
				Type:         Json,
			}
		}
		data = typedError
		w.WriteHeader(typedError.Status)
	} else {
		w.WriteHeader(http.StatusOK) // status is ok
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("Error encoding data : %v", err)
	}
}

type previewRequest struct {
	Template      string
	Source        string
	SourceChanged bool
}

type previewResponse struct {
	Result string `json:"result"`
}

func strooHandler(command *Command) http.HandlerFunc {

	var cachedResult *Code

	return func(w http.ResponseWriter, r *http.Request) {
		var request previewRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			respond(w, err)
			return
		}

		// template is more likely to change : we're processing it first
		tmpTemplate, err := template.New(packageName).Funcs(DefaultFuncMap()).Parse(request.Template)
		if err != nil {
			respond(w, InvalidTemplate2, err.Error())
			return
		}

		// we're using cached result, so we don't stress the disk for nothing
		if request.SourceChanged || cachedResult == nil {
			// first we check the correctness of the source, so we don't write down for nothing
			fileSet := token.NewFileSet()
			_, err := parser.ParseFile(fileSet, packageName, request.Source, parser.DeclarationErrors|parser.AllErrors)
			if err != nil {
				respond(w, InvalidGoSource, err.Error())
				return
			}

			// prepare a temp project
			tempProj, err := CreateTempProj([]TemporaryPackage{{Name: packageName, Files: map[string]interface{}{"file.go": request.Source}}})
			if err != nil {
				respond(w, InvalidGoSource, err.Error())
				return
			}
			// setup cleanup, so temporary files and folders gets deleted
			defer tempProj.Cleanup()

			tempProj.Config.Mode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedImports
			// load package using the old way
			thePackages, err := packages.Load(tempProj.Config, fmt.Sprintf("file=%s", tempProj.File(packageName, "file.go")))
			if err != nil {
				respond(w, InvalidPackage, err.Error())
				return
			}
			if len(thePackages) != 1 {
				respond(w, MalformedRequest{ErrorMessage: "expecting exactly one package", Type: OnePackage})
				return
			}

			// create a temporary command to analyse the loaded package
			codeBuilder := DefaultAnalyzer()
			tempCommand := NewCommand(codeBuilder)
			tempCommand.TestMode = command.TestMode
			tempCommand.DebugPrint = command.DebugPrint
			if err := tempCommand.Analyse(codeBuilder, thePackages[0]); err != nil {
				respond(w, InvalidAnalysis, err.Error())
				return
			}
			// convention : by default, the upper most type struct is provided to the code builder
			var firstType *TypeInfo
			if len(tempCommand.Result.Types) >= 1 {
				firstType = tempCommand.Result.Types[0]
			}
			// create code
			cachedResult, err = New(tempCommand.Result, tempCommand.CodeConfig, nil, "")
			if err != nil {
				respond(w, MalformedRequest{ErrorMessage: err.Error()})
				return
			}
			if firstType != nil {
				cachedResult.Main, err = cachedResult.StructByKey(firstType.Name)
				if err != nil {
					respond(w, MalformedRequest{ErrorMessage: err.Error()})
					return
				}
			}
		}
		// set the template to the result (might have been changed)
		cachedResult.tmpl = tmpTemplate
		cachedResult.ResetKeeper() // reset kept data (so we can refill it)

		// finally, we're processing the template over the result
		var buf bytes.Buffer
		if err := tmpTemplate.Execute(&buf, cachedResult); err != nil {
			respond(w, InvalidTemplate, err.Error())
			return
		}
		optImports, err := imports.Process(packageName, buf.Bytes(), nil)
		if err != nil {
			respond(w, InvalidFormat, err.Error(), buf.String())
			return
		}

		formatted, err := format.Source(optImports)
		if err != nil {
			respond(w, InvalidFormat, err.Error(), string(optImports))
			return
		}

		response := previewResponse{Result: string(formatted)}
		respond(w, response)
	}
}

func StartPlayground(command *Command) {
	log.Printf("Starting on http://localhost:8080\n")
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could NOT obtain current workdir : %v", err)
	}
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}
	router := mux.NewRouter()

	router.NotFoundHandler = filesHandler(wd, statikFS, command.TestMode)
	router.HandleFunc("/stroo-it", strooHandler(command)).Methods("POST")
	if err := http.ListenAndServe("0.0.0.0:8080", router); err != nil {
		log.Fatalf("error while serving : %v", err)
	}
	log.Printf("Server started.")
}
