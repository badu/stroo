package stroo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"go/format"
	"golang.org/x/tools/go/packages"
	"log"
	"net/http"
	"os"
	"text/template"
)

const (
	playgroundVersion = "0.0.1"
	storageFolder     = "/server"
	indexHTML         = "/index.html"
	codeTextAreaHTML  = "/codetextarea.html"
	playgroundHTML    = "/playground.html"
	favico            = "/favicon.ico"
	exampleGo         = "/example-source.go"
	exampleTemplate   = "/example-template.tpl"
)

func exampleSourceHandler(workingDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, workingDir+storageFolder+exampleGo)
	}
}

func exampleTemplateHandler(workingDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, workingDir+storageFolder+exampleTemplate)
	}
}

func filesHandler(workingDir string) http.HandlerFunc {
	// prepare index.html
	index, err := template.ParseFiles(workingDir + storageFolder + indexHTML)
	if err != nil {
		log.Fatalf("Error parsing template : %v\n working dir is = %q", err, workingDir)
	}
	type pageinfo struct {
		Version string
	}
	info := pageinfo{
		Version: playgroundVersion,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case indexHTML, "/":
			if err := index.ExecuteTemplate(w, "index", info); err != nil {
				log.Printf("error producing index template : %v", err)
				return
			}
		case codeTextAreaHTML, playgroundHTML:
			http.ServeFile(w, r, workingDir+storageFolder+"/"+r.URL.Path)
		case favico:
			// just ignore it
		default:
			log.Printf("Requested UNKNOWN URL : %q", r.URL.Path)
		}
	})
}

func respond(w http.ResponseWriter, _ *http.Request, data interface{}, code int) {
	if err, ok := data.(error); ok {
		log.Printf("Error : %v", err)
		data = struct {
			Error string `json:"error"`
		}{Error: err.Error()}
	}

	//if errors.Is()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("Error encoding data : %v", err)
	}
}

func strooHandler() http.HandlerFunc {

	type previewRequest struct {
		Template string
		Source   string
	}

	type previewResponse struct {
		Result string `json:"output"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var request previewRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			respond(w, r, fmt.Errorf("error decoding data : %v", err), http.StatusBadRequest)
			return
		}
		// template is more likely to change : we're processing it first
		tmpTemplate, err := template.New("template").Funcs(DefaultFuncMap()).Parse(request.Template)
		if err != nil {
			respond(w, r, fmt.Errorf("error parsing template : %v", err), http.StatusBadRequest)
			return
		}
		// prepare a temp project
		tempProj, err := CreateTempProj([]TemporaryPackage{{Name: "playground", Files: map[string]interface{}{"file.go": request.Source}}})
		if err != nil {
			respond(w, r, fmt.Errorf("failed creating temporary project : %v", err), http.StatusBadRequest)
			return
		}
		// setup cleanup, so temporary files and folders gets deleted
		defer tempProj.Cleanup()

		tempProj.Config.Mode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedImports
		// load package using the old way
		thePackages, err := packages.Load(tempProj.Config, fmt.Sprintf("file=%s", tempProj.File("playground", "file.go")))
		if err != nil {
			respond(w, r, fmt.Errorf("failed loading packages : %v", err), http.StatusBadRequest)
			return
		}
		if len(thePackages) != 1 {
			respond(w, r, errors.New("%d packages found. should be exactly one"), http.StatusBadRequest)
			return
		}
		// create a temporary command to analyse the loaded package
		tempCommand := NewCommand(DefaultAnalyzer())
		if err := tempCommand.Analyse(thePackages[0]); err != nil {
			respond(w, r, fmt.Errorf("error analyzing package : %v", err), http.StatusBadRequest)
			return
		}
		// convention : by default, the upper most type struct is provided to the code builder
		var firstType *TypeInfo
		if len(tempCommand.Result.Types) >= 1 {
			firstType = tempCommand.Result.Types[0]
		} else {
			respond(w, r, errors.New("no types found. please a a type"), http.StatusBadRequest)
			return
		}
		// create code
		result := Code{
			PackageInfo: tempCommand.Result,
			CodeConfig:  tempCommand.CodeConfig,
			keeper:      make(map[string]interface{}),
			tmpl:        tmpTemplate,
			Main:        TypeWithRoot{T: firstType},
		}
		result.Main.D = &result
		// finally, we're processing the template over the result
		var buf bytes.Buffer
		if err := tmpTemplate.Execute(&buf, &result); err != nil {
			respond(w, r, fmt.Errorf("error executing template : %v", err), http.StatusBadRequest)
			return
		}
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			respond(w, r, fmt.Errorf("go/format error: %v\nGo source:\n%s", err, buf.String()), http.StatusBadRequest)
			return
		}

		response := previewResponse{Result: string(formatted)}
		respond(w, r, response, http.StatusOK)
	}
}

func StartPlayground(command *Command) {
	log.Printf("Starting on http://localhost:8080\n")
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could NOT obtain current workdir : %v", err)
	}
	router := mux.NewRouter()
	router.Handle("/example-source", exampleSourceHandler(wd)).Methods("GET")
	router.Handle("/example-template", exampleTemplateHandler(wd)).Methods("GET")
	router.NotFoundHandler = filesHandler(wd)
	router.HandleFunc("/stroo-it", strooHandler()).Methods("POST")
	if err := http.ListenAndServe("0.0.0.0:8080", router); err != nil {
		log.Fatalf("error while serving : %v", err)
	}
	log.Printf("Server started.")
}
