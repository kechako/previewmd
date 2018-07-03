package main

import (
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"sync"
)

type CSSHandler struct {
	Name string

	ctype   string
	content []byte
	once    sync.Once
}

func (h *CSSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(h.load)

	w.Header().Add("Content-Type", h.ctype)
	_, err := w.Write(h.content)
	if err != nil {
		log.Printf("Error : %v", err)
	}
}

func (h *CSSHandler) load() {
	h.ctype = mime.TypeByExtension(".css")

	var r io.Reader
	if h.Name == "" {
		file, err := Assets.Open("/css/github-markdown.css")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		r = file
	} else {
		file, err := os.Open(h.Name)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		r = file
	}

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	h.content = buf
}
