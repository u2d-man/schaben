package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
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
	hClient http.Client
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

	f, err := os.Create("urls.txt")
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

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
				_, err := f.Write([]byte(aURL + "\n"))
				if err != nil {
					return false
				}
			}

			return true
		})

		return true
	})

	return ExitCodeOK
}

// Retrieve content such as article body and title.
func (c *CLI) articleContentExtractor(crawlerSite CrawlerSite) int {
	f, err := os.Open("urls.txt")
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	urls := make([]string, 0, 120)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	if err = scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	for _, url := range urls {
		doc, err := scraping(url)
		if err != nil {
			_, _ = fmt.Fprintln(c.errStream, err.Error())
			return ExitCodeFail
		}

		// Remove unneeded classes.
		removed := doc.RemoveClass(crawlerSite.RemoveClass)

		title := removed.Find(crawlerSite.Title).Text()
		body := removed.Find(crawlerSite.Body).Text()
		articleUpdatedAt := doc.Find(crawlerSite.ArticleUpdatedAt).Text()

		fmt.Println(url)
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
