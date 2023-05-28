package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
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

var (
	db  *sqlx.DB
	err error
)

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
