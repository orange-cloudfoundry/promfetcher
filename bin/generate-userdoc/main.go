package main

import (
	"bytes"
	"github.com/gobuffalo/packr/v2"
	"os"
	"text/template"
)

var boxTemplates = packr.New("userdocs_templates", "../../templates")

func main() {
	buf := &bytes.Buffer{}
	userdoc, _ := boxTemplates.FindString("how-to-use.md")
	tpl, err := template.New("").Parse(userdoc)
	if err != nil {
		panic(err)
	}
	err = tpl.Execute(buf, struct {
		BaseURL string
	}{"my.promfetcher.com"})
	os.Stdout.Write(buf.Bytes())
	if err != nil {
		panic(err)
	}
}
