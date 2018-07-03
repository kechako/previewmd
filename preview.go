package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

type PreviewHandler struct {
	Markdown *Markdown

	template *template.Template
	once     sync.Once
}

type MarkdownData struct {
	HTML string `json:"html"`
	Hash string `json:"hash"`
}

func (h *PreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(h.loadTemplate)

	data := &MarkdownData{
		HTML: h.Markdown.HTML,
		Hash: h.Markdown.Hash,
	}

	var buf bytes.Buffer
	err := h.template.Execute(&buf, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "text/html")

	io.Copy(w, &buf)
}

const templateName = "/templates/preview.tmpl"

func (h *PreviewHandler) loadTemplate() {
	file, err := Assets.Open(templateName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	h.template = template.Must(template.New(templateName).Funcs(template.FuncMap{
		"safeurl":  func(u string) template.URL { return template.URL(u) },
		"safehtml": func(u string) template.HTML { return template.HTML(u) },
	}).Parse(string(buf)))
}

type ModifiedHandler struct {
	Markdown *Markdown
}

func (h *ModifiedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := &MarkdownData{
		Hash: h.Markdown.Hash,
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	io.Copy(w, &buf)
}
