package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/jmoiron/sqlx"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ExitCodeOK   = 0
	ExitCodeFail = 1
	ArgsFilename = 1
)

type CLI struct {
	outStream, errStream io.Writer
}

var (
	db      *sqlx.DB
	hClient http.Client
	err     error
)

type Targets struct {
	CrawlTarget CrawlerSite `json:"target"`
}

type CrawlerSite struct {
	Domain               string `json:"domain"`
	URL                  string `json:"url"`
	Block                string `json:"block"`
	ArticleLinkFromBlock string `json:"article_link_from_block"`
	Title                string `json:"title"`
	Body                 string `json:"body"`
	ArticleUpdatedAt     string `json:"article_updated_at"`
	RemoveClass          string `json:"remove_class"`
}

type ArticleURL struct {
	ID  int    `db:"id"`
	URL string `db:"url"`
}

func NewCLI(outStream, errStream io.Writer) *CLI {
	return &CLI{outStream: outStream, errStream: errStream}
}

func init() {
	hClient = http.Client{
		Timeout: 5 * time.Second,
	}
}

func main() {
	cmd := NewCLI(os.Stdout, os.Stderr)
	os.Exit(cmd.execute(os.Args))
}

func (c *CLI) execute(args []string) int {
	var filename string

	if len(args) == ArgsFilename {
		panic("specify the file name as the first argument.")
	}
	for i, v := range args {
		if i == 1 {
			filename = v
		}
	}

	f, err := os.Open(filename)
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	b, err := io.ReadAll(f)
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	fmt.Println(string(b))

	var targets Targets
	if err = json.Unmarshal(b, &targets); err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	fmt.Println(targets)

	crawlTarget := targets.CrawlTarget

	if crawlTarget.Domain == "" {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	c.articleURLRetriever(crawlTarget)
	c.articleContentExtractor(crawlTarget)

	return ExitCodeOK
}

// Get url from the top page of the site.
func (c *CLI) articleURLRetriever(crawlerSite CrawlerSite) int {
	doc, err := scraping(crawlerSite.URL)

	// pickup article urls
	doc.Find(crawlerSite.Block).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		s.Find(crawlerSite.ArticleLinkFromBlock).EachWithBreak(func(i int, s *goquery.Selection) bool {
			aURL, exists := s.Attr("href")
			if exists != true {
				_, _ = fmt.Fprintln(c.errStream, err.Error())
				return false
			}

			// fragment check
			if !strings.Contains(aURL, "#") {
				fmt.Println(aURL)
			}

			return true
		})

		return true
	})

	return ExitCodeOK
}

// Retrieve content such as article body and title.
func (c *CLI) articleContentExtractor(crawlerSite CrawlerSite) int {
	var articleURLs []ArticleURL

	for _, articleURL := range articleURLs {
		doc, err := scraping(articleURL.URL)
		if err != nil {
			_, _ = fmt.Fprintln(c.errStream, err.Error())
			return ExitCodeFail
		}

		// Remove unneeded classes.
		removed := doc.RemoveClass(crawlerSite.RemoveClass)

		title := removed.Find(crawlerSite.Title).Text()
		body := removed.Find(crawlerSite.Body).Text()
		articleUpdatedAt := doc.Find(crawlerSite.ArticleUpdatedAt).Text()

		fmt.Println(articleURL.URL)
		fmt.Println(strings.ReplaceAll(title, "\n", ""))
		fmt.Println(body)
		fmt.Println(articleUpdatedAt)

		fmt.Println("sleep")
		time.Sleep(2 * time.Second)
	}

	return ExitCodeOK
}

func scraping(URL string) (*goquery.Document, error) {
	resp, err := hClient.Get(URL)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			_ = fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
		}
	}(resp.Body)

	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}
