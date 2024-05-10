package main

import (
	"bytes"
	_ "embed"
	"os"
	"text/template"
)

//go:embed how-to-use.md
var howToUse []byte

func main() {
	buf := &bytes.Buffer{}
	tpl, err := template.New("").Parse(string(howToUse))
	if err != nil {
		panic(err)
	}
	err = tpl.Execute(buf, struct {
		BaseURL string
	}{"my-promfetcher.example.net"})
	if err != nil {
		panic(err)
	}
	if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
		panic(err)
	}
}
