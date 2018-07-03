package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
)

type PreviewHandler struct {
	Markdown *Markdown

	fileDir     string
	previewPath string

	fileServer http.Handler
	template   *template.Template
}

func NewPreviewHandler(markdown *Markdown) *PreviewHandler {
	fileDir := filepath.Dir(markdown.Name)
	h := &PreviewHandler{
		Markdown:    markdown,
		fileDir:     fileDir,
		previewPath: path.Join("/", filepath.Base(markdown.Name)),
		fileServer:  http.FileServer(http.Dir(fileDir)),
	}
	h.loadTemplate()

	return h
}

type MarkdownData struct {
	HTML string `json:"html"`
	Hash string `json:"hash"`
}

func (h *PreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != h.previewPath {
		h.fileServer.ServeHTTP(w, r)
		return
	}

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
