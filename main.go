package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-sql-driver/mysql"
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
)

type CLI struct {
	outStream, errStream io.Writer
}

var (
	db      *sqlx.DB
	hClient http.Client
	err     error
)

type CrawlerSite struct {
	CrawlerSiteID        int    `db:"crawler_site_id"`
	CrawlerSiteSettingID int    `db:"crawler_site_setting_id"`
	Domain               string `db:"domain"`
	URL                  string `db:"url"`
	Block                string `db:"block"`
	ArticleLinkFromBlock string `db:"article_link_from_block"`
	Title                string `db:"title"`
	Body                 string `db:"body"`
	RemoveClass          string `db:"remove_class"`
	ArticleUpdatedAt     string `db:"article_updated_at"`
}

type ArticleURL struct {
	ID  int    `db:"id"`
	URL string `db:"url"`
}

func getEnv(key string, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}

func connectDB() (*sqlx.DB, error) {
	config := mysql.NewConfig()
	config.Net = "tcp"
	config.Addr = getEnv("DB_HOST", "127.0.0.1") + ":" + getEnv("DB_PORT", "3306")
	config.User = getEnv("DB_USER", "davy_elton")
	config.Passwd = getEnv("DB_PASSWORD", "password")
	config.DBName = getEnv("DB_NAME", "schaben_local")
	config.ParseTime = true
	dsn := config.FormatDSN()

	return sqlx.Open("mysql", dsn)
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
	os.Exit(cmd.execute())
}

func (c *CLI) execute() int {
	db, err = connectDB()
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			_, _ = fmt.Fprintln(c.errStream, err.Error())
		}
	}(db)

	var crawlerSite []CrawlerSite
	query := "SELECT `cs`.`id` as `crawler_site_id`, " +
		"`css`.`id` as `crawler_site_setting_id`, " +
		"`cs`.`domain`, " +
		"`cs`.`url`, " +
		"`css`.`block`, " +
		"`css`.`article_link_from_block`, " +
		"`css`.`title`, " +
		"`css`.`body`, " +
		"`css`.`remove_class`," +
		"`css`.`article_updated_at` " +
		"FROM `crawler_site` as `cs` " +
		"JOIN `crawler_site_setting` as `css` ON (`cs`.`id` = `css`.`crawler_site_id`) "
	if err := db.Select(&crawlerSite, query); err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	c.articleURLRetriever(crawlerSite)
	c.articleContentExtractor(crawlerSite)

	return ExitCodeOK
}

// Get url from the top page of the site.
func (c *CLI) articleURLRetriever(crawlerSite []CrawlerSite) int {
	doc, err := scraping(crawlerSite[1].URL)

	// pickup article urls
	doc.Find(crawlerSite[1].Block).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		s.Find(crawlerSite[1].ArticleLinkFromBlock).EachWithBreak(func(i int, s *goquery.Selection) bool {
			aURL, exists := s.Attr("href")
			if exists != true {
				_, _ = fmt.Fprintln(c.errStream, err.Error())
				return false
			}

			// fragment check
			if !strings.Contains(aURL, "#") {
				var countURL int
				// duplicate check
				err = db.Get(&countURL, "SELECT count(*) FROM `article_url` WHERE `url` = ?", aURL)
				if err != nil {
					_, _ = fmt.Fprintln(c.errStream, err.Error())
					return false
				}

				if countURL == 0 {
					_, err = db.Exec("INSERT INTO `article_url` "+
						"(`crawler_site_id`, `crawler_site_setting_id`, `url`) VALUES (?, ?, ?)",
						crawlerSite[1].CrawlerSiteID, crawlerSite[1].CrawlerSiteSettingID, aURL)
					if err != nil {
						_, _ = fmt.Fprintln(c.errStream, err.Error())
						return false
					}
				}
			}

			return true
		})

		return true
	})

	return ExitCodeOK
}

// Retrieve content such as article body and title.
func (c *CLI) articleContentExtractor(crawlerSite []CrawlerSite) int {
	var articleURLs []ArticleURL
	query := "SELECT `id`, `url` FROM `article_url` LIMIT 5"
	err = db.Select(&articleURLs, query)
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	for _, articleURL := range articleURLs {
		doc, err := scraping(articleURL.URL)

		// Remove unneeded classes.
		removed := doc.RemoveClass(crawlerSite[1].RemoveClass)

		title := removed.Find(crawlerSite[1].Title).Text()
		body := removed.Find(crawlerSite[1].Body).Text()
		articleUpdatedAt := doc.Find(crawlerSite[1].ArticleUpdatedAt).Text()

		fmt.Println(articleURL.URL)
		fmt.Println(strings.ReplaceAll(title, "\n", ""))
		fmt.Println(body)
		fmt.Println(articleUpdatedAt)

		_, err = db.Exec("INSERT INTO `archive` "+
			"(`crawler_site_id`, `url`, `title`, `body`, `article_updated_at`) VALUES (?, ?, ?, ?, ?)",
			crawlerSite[1].CrawlerSiteID, articleURL.URL, strings.ReplaceAll(title, "\n", ""), body, articleUpdatedAt)
		if err != nil {
			_, _ = fmt.Fprintln(c.errStream, err.Error())
			return ExitCodeFail
		}

		_, err = db.Exec("DELETE FROM `article_url` WHERE `id` = ?", articleURL.ID)
		if err != nil {
			_, _ = fmt.Fprintln(c.errStream, err.Error())
			return ExitCodeFail
		}

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
