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
	db  *sqlx.DB
	err error
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
	ArticleUpdatedAt     string `db:"article_updated_at"`
}

type ArticleURL struct {
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
		"`css`.`article_updated_at` " +
		"FROM `crawler_site` as `cs` " +
		"JOIN `crawler_site_setting` as `css` ON (`cs`.`id` = `css`.`crawler_site_id`) "
	if err := db.Select(&crawlerSite, query); err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	resp, err := requestSite(crawlerSite[1].URL)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			_ = fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
		}
	}(resp.Body)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	fmt.Println("get doc")

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
				fmt.Println(aURL)

				var countURL int
				// duplicate check
				err = db.Get(&countURL, "SELECT count(*) FROM `article_url` WHERE `url` = ?", aURL)
				if err != nil {
					_, _ = fmt.Fprintln(c.errStream, err.Error())
					return false
				}

				fmt.Println(countURL == 0)

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

	var articleURLs []ArticleURL
	query = "SELECT `url` FROM `article_url` LIMIT 1"
	err = db.Select(&articleURLs, query)
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	for _, articleURL := range articleURLs {
		resp, err = requestSite(articleURL.URL)
		time.Sleep(2)

		doc, err = goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			_, _ = fmt.Fprintln(c.errStream, err.Error())
			return ExitCodeFail
		}

		title := doc.Find(crawlerSite[1].Title).Text()
		body := doc.Find(crawlerSite[1].Body).Text()

		fmt.Println(title)
		fmt.Println(body)
	}

	return ExitCodeOK
}

func requestSite(URL string) (*http.Response, error) {
	resp, err := http.Get(URL)
	if err != nil {
		return nil, err
	}

	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		return resp, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	return resp, nil
}
