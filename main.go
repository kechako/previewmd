package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/browser"
)

var (
	markdownName    string
	httpAddr        string
	useGitHub       bool
	markdownContext string
	cssFile         string
)

const version = "1.0"

func printVersion() {
	fmt.Printf("%s %s\n", flag.CommandLine.Name(), version)
	os.Exit(0)
}

func parseFlags() {
	var ver bool
	flag.StringVar(&httpAddr, "addr", ":8080", "http address to listen.")
	flag.BoolVar(&useGitHub, "github", true, "use GitHub API to generate markdown.")
	flag.StringVar(&markdownContext, "context", "", "context of markdown.")
	flag.StringVar(&cssFile, "css", "", "CSS path to use in HTML.")
	flag.BoolVar(&ver, "v", false, "show version.")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage: %s markdown

Parameters:
  markdown
        markdown file to preview.

Options:
`, flag.CommandLine.Name())
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()

	if ver {
		printVersion()
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error : Markdown file is not specified.")
		os.Exit(2)
	}

	var err error
	markdownName, err = filepath.Abs(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error : Markdown filename is not valid.")
		os.Exit(2)
	}

	if _, err := os.Stat(markdownName); err != nil {
		fmt.Fprintln(os.Stderr, "Error : Markdown file is not found.")
		os.Exit(1)
	}

	if cssFile != "" {
		cssFile, err = filepath.Abs(cssFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error : CSS filename is not valid.")
			os.Exit(2)
		}

		if _, err := os.Stat(cssFile); err != nil {
			fmt.Fprintln(os.Stderr, "Error : CSS file is not found.")
			os.Exit(1)
		}
	}
}

func previewURL(addr string) (string, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return "", err
	}

	var host string
	if len(tcpAddr.IP) == 0 || tcpAddr.IP.IsLoopback() || tcpAddr.IP.IsUnspecified() {
		host = "localhost"
	} else {
		host = tcpAddr.IP.String()
	}

	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, strconv.Itoa(tcpAddr.Port)),
	}).String(), nil
}

func main() {
	parseFlags()

	markdown, err := NewMarkdown(markdownName, useGitHub, markdownContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error : %v\n", err)
		os.Exit(1)
	}
	defer markdown.Close()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ch
		cancel()
	}()

	mux := http.NewServeMux()
	mux.Handle("/css/github-markdown.css", &CSSHandler{Name: cssFile})
	mux.Handle("/modified", &ModifiedHandler{Markdown: markdown})
	mux.Handle("/", NewPreviewHandler(markdown))

	srv := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error : %v\n", err)
		}
	}()

	l, err := net.Listen("tcp", httpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error : %v\n", err)
		os.Exit(1)
	}

	u, err := previewURL(httpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error : %v\n", err)
		os.Exit(1)
	}
	log.Printf("Listen to %s ...", httpAddr)
	log.Printf("Open brower and visit %s...", u)
	browser.OpenURL(u)

	err = srv.Serve(l)
	if err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Error : %v\n", err)
		os.Exit(1)
	}
}
