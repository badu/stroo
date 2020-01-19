package stroo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"os"
)

type previewRequest struct {
	Template string
	Source   string
}

type previewResponse struct {
	Output string `json:"output"`
}

func strooHandler(w http.ResponseWriter, r *http.Request) {
	var req previewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding data : %v", err)
		return
	}
	log.Printf("Received : %#v\n", req)
	//codeSource := source.Reader("source.go", strings.NewReader(req.Source))
	//templateSource := source.Reader("template.tpl", strings.NewReader(req.Template))
	//j := tool.Job{
	//	Code:     codeSource,
	//	Template: templateSource,
	//}
	var buf bytes.Buffer
	//if err := tmpl.Execute(&buf); err != nil {
	//	log.Printf("Error processing template : %v", err)
	//	respond(w, r, err, http.StatusBadRequest)
	//	return
	//}
	buf.WriteString(req.Source)
	res := previewResponse{
		Output: buf.String(),
	}
	fmt.Sprintf("%#v", res)
	//respond(w, r, errors.New("test error"), http.StatusBadRequest)
	respond(w, r, res, http.StatusOK)
}

func exampleSourceHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./server/example-source.go")
}

func exampleTemplateHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./server/example-template.tpl")
}

func respond(w http.ResponseWriter, r *http.Request, data interface{}, code int) {
	if err, ok := data.(error); ok {
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

func filesHandler() http.HandlerFunc {
	const Current = "0.0.1"
	wd, _ := os.Getwd()
	index, err := template.ParseFiles(wd + "/server/index.html")
	if err != nil {
		log.Printf("Error parsing template : %v\n working dir is = %q", err, wd)
		os.Exit(1)
	}

	type pageinfo struct {
		Version string
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			return
		}
		switch r.URL.Path {
		case "/index.html", "/":
			info := pageinfo{
				Version: Current,
			}
			if err := index.ExecuteTemplate(w, "index", info); err != nil {
				log.Printf("Error exe tmpl layout : %v", err)
				return
			}
		case "/codetextarea.html", "/playground.html":
			http.ServeFile(w, r, wd+"/server/"+r.URL.Path)
		case "/favicon.ico":
			// just ignore it
		default:
			log.Printf("Requested URL unknown: %q", r.URL.Path)
		}
	})
}

func StartPlayground() {
	log.Printf("Starting on http://localhost:8080\n")

	router := mux.NewRouter()
	router.Handle("/stroo-it", http.HandlerFunc(strooHandler)).Methods("POST")
	router.Handle("/example-source", http.HandlerFunc(exampleSourceHandler)).Methods("GET")
	router.Handle("/example-template", http.HandlerFunc(exampleTemplateHandler)).Methods("GET")
	router.NotFoundHandler = filesHandler()
	err := http.ListenAndServe("0.0.0.0:8080", router)
	if err != nil {
		log.Fatalf("Error while serving : %v", err)
	}
	log.Printf("Server started.")
}
