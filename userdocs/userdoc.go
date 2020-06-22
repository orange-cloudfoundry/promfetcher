package userdocs

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gobuffalo/packr/v2"
)

type UserDoc struct {
	baseUrl string
}

var boxTemplates = packr.New("userdocs_templates", "./templates")
var mainTpl *template.Template

func NewUserDoc(baseUrl string) *UserDoc {
	var err error
	mainFile, _ := boxTemplates.FindString("main.html")
	mainTpl, err = template.New("main.html").Funcs(tplfuncs).Parse(mainFile)
	if err != nil {
		panic(fmt.Sprintf("Cannot parse template 'main.html': %s", err.Error()))
	}
	for _, tplName := range boxTemplates.List() {
		if tplName == "main.html" {
			continue
		}
		tplTxt, _ := boxTemplates.FindString(tplName)
		_, err := mainTpl.New(tplName).Funcs(tplfuncs).Parse(tplTxt)
		if err != nil {
			panic(fmt.Sprintf("Cannot parse template '%s': %s", tplName, err.Error()))
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
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(buf.Bytes())
}
