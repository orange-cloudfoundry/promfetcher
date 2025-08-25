package userdocs

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

type UserDoc struct {
	baseUrl        string
	EmbededUserDoc fs.FS
}

var mainTpl *template.Template

// EmbededUserDoc holds our static data
//
//go:embed templates assets
var EmbededUserDoc embed.FS

func NewUserDoc(baseUrl string) *UserDoc {
	var err error
	mainFile, _ := EmbededUserDoc.ReadFile("main.html")
	mainTpl, err = template.New("main.html").Funcs(tplfuncs).Parse(string(mainFile))
	if err != nil {
		panic(fmt.Sprintf("Cannot parse template 'main.html': %s", err.Error()))
	}
	content, err := EmbededUserDoc.ReadDir(".")
	if err != nil {
		panic(err)
	}
	for _, entry := range content {
		if entry.Name() == "main.html" {
			continue
		}
		tpl, _ := EmbededUserDoc.ReadFile(entry.Name())
		_, err := mainTpl.New(entry.Name()).Funcs(tplfuncs).Parse(string(tpl))
		if err != nil {
			panic(fmt.Sprintf("Cannot parse template '%s': %s", entry, err.Error()))
		}
	}
	return &UserDoc{
		baseUrl: baseUrl,
	}
}

func (d UserDoc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	err := mainTpl.Execute(buf, struct {
		BaseURL string
	}{d.baseUrl})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(buf.Bytes())
}
