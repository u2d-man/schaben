package main

import (
	"flag"
	"fmt"
	"io"
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

	var tURL string
	flags.StringVar(&tURL, "u", "", "scraping target url")

	err := flags.Parse(args[1:])
	if err != nil {
		return ExitCodeParseFlagError
	}

	return run(tURL)
}

func run(tURL string) int {
	doc, err := goquery.NewDocument(tURL)
	if err != nil {
		panic(err)
	}

	fmt.Println(doc)

	return 0
}
