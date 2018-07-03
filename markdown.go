package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"html/template"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

type Markdown struct {
	Name    string
	Context string

	HTML string
	Hash string

	watcher *fsnotify.Watcher
	done    context.CancelFunc

	client   *github.Client
	renderer blackfriday.Renderer
}

func NewMarkdown(name string, useGitHub bool) (*Markdown, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize markdown watcher")
	}
	watcher.Add(filepath.Dir(name))

	ctx, cancel := context.WithCancel(context.Background())

	m := &Markdown{
		Name:    name,
		watcher: watcher,
		done:    cancel,
	}

	if useGitHub {
		m.client = github.NewClient(nil)
	} else {
		m.renderer = blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
			Flags: blackfriday.HTMLFlagsNone,
		})
	}

	m.Generate()

	go m.watch(ctx)

	return m, nil
}

func (m *Markdown) Close() error {
	m.done()
	return m.watcher.Close()
}

func (m *Markdown) watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-m.watcher.Events:
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 &&
				event.Name == m.Name {
				m.Generate()
			}
		case err := <-m.watcher.Errors:
			log.Println("error : ", err)
		}
	}
}

func (m *Markdown) Generate() {
	text, err := ioutil.ReadFile(m.Name)
	if err != nil {
		m.HTML = errorHTML(err)
		return
	}

	hash := sha1Sum(text)
	if hash == m.Hash {
		// not changed
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var html string
	if m.client != nil {
		html, err = m.markdownWithGitHub(ctx, string(text), "")
	} else {
		html, err = m.markdownWithBlackFriday(ctx, string(text), "")
	}
	if err != nil {
		m.HTML = errorHTML(err)
		return
	}

	m.HTML = html
	m.Hash = hash
}

func (m *Markdown) markdownWithGitHub(ctx context.Context, text string, context string) (string, error) {
	html, _, err := m.client.Markdown(ctx, text, &github.MarkdownOptions{
		Mode:    "gfm",
		Context: m.Context,
	})
	if err != nil {
		return "", err
	}

	return html, nil
}

func (m *Markdown) markdownWithBlackFriday(ctx context.Context, text string, context string) (string, error) {
	html := blackfriday.Run(
		[]byte(text),
		blackfriday.WithExtensions(blackfriday.CommonExtensions),
		blackfriday.WithRenderer(m.renderer))

	return string(html), nil
}

func sha1Sum(text []byte) string {
	hash := sha1.Sum(text)
	return hex.EncodeToString(hash[:])
}

func errorHTML(err error) string {
	return "<p> Error : " + template.HTMLEscapeString(err.Error()) + "</p>"
}
