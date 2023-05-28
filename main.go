package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
)

const (
	ExitCodeOK             = 0
	ExitCodeParseFlagError = 1
	ExitCodeFail           = 1
)

type CLI struct {
	outStream, errStream io.Writer
}

func NewCLI(outStream, errStream io.Writer) *CLI {
	return &CLI{outStream: outStream, errStream: errStream}
}

func main() {
	cmd := NewCLI(os.Stdout, os.Stderr)
	os.Exit(cmd.Execute(os.Args))
}

func (c *CLI) Execute(args []string) int {
	flags := flag.NewFlagSet("schaben", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var url string
	flags.StringVar(&url, "u", "", "scraping target url")

	err := flags.Parse(args[1:])
	if err != nil {
		return ExitCodeParseFlagError
	}

	return c.run(url)
}

func (c *CLI) run(url string) int {
	resp, err := http.Get(url)
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			_, _ = fmt.Fprintln(c.errStream, err.Error())
		}
	}(resp.Body)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	sec := doc.Find("h1")
	sec.Each(func(i int, s *goquery.Selection) {
		fmt.Println(s.Text())
	})

	return ExitCodeOK
}
