package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"os"
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
	os.Exit(cmd.execute())
}

func (c *CLI) execute() int {
	resp, err := http.Get("")
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

//func (c *CLI) run() int {
//	resp, err := http.Get(url)
//	if err != nil {
//		_, _ = fmt.Fprintln(c.errStream, err.Error())
//		return ExitCodeFail
//	}
//
//	defer func(Body io.ReadCloser) {
//		err := Body.Close()
//		if err != nil {
//			_, _ = fmt.Fprintln(c.errStream, err.Error())
//		}
//	}(resp.Body)
//
//	doc, err := goquery.NewDocumentFromReader(resp.Body)
//	if err != nil {
//		_, _ = fmt.Fprintln(c.errStream, err.Error())
//		return ExitCodeFail
//	}
//
//	sec := doc.Find("h1")
//	sec.Each(func(i int, s *goquery.Selection) {
//		fmt.Println(s.Text())
//	})
//
//	return ExitCodeOK
//}
