package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
	"sync"
)

var (
	mu sync.Mutex
	dup = map[string]bool{}
	spreadsheets = map[string]bool{}
	totalProcessed = 0
)

func main() {
	if len(os.Args) < 2 {
		fmt.Errorf("Must provide a root url\n")
		return
	}

	rootUrlStr := os.Args[1]
	rootUrl, err := url.Parse(rootUrlStr)
	if err != nil {
		fmt.Errorf("Invalid root url: %s\n", err)
	}
	dup[rootUrl.Path] = true

	mux := fetchbot.NewMux()

	mux.HandleErrors(fetchbot.HandlerFunc(func(ctx *fetchbot.Context, res *http.Response, err error) {
		fmt.Printf("[ERR] %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
	}))

	questionRegex, err := regexp.Compile(`^/questions/\d+`)
	if err != nil {
		log.Panic(err)
	}

	mux.Response().Method("GET").ContentType("text/html").Handler(fetchbot.HandlerFunc(
		func(ctx *fetchbot.Context, res *http.Response, err error) {
			//fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())
			doc, err := goquery.NewDocumentFromResponse(res)
			if err != nil {
				fmt.Printf("[ERR] %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
				return
			}
			doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
				val, _ := s.Attr("href")
				u, err := ctx.Cmd.URL().Parse(val)
				if err != nil {
					return
				}

				urlStr := u.String()
				if strings.HasSuffix(urlStr, "/") {
					urlStr = urlStr[0:len(urlStr)-1]
				}

				if strings.HasSuffix(urlStr, ".png") || strings.HasSuffix(urlStr, ".jpg") {
					return
				}

				if u.Host == "bl.ocks.org" {
					fmt.Printf("%s\t%s\n", urlStr, ctx.Cmd.URL())
				} else if u.Host == rootUrl.Host && (u.Path == "/search" || questionRegex.MatchString(u.Path)) {
					mu.Lock()
					if !dup[u.Path] {
						dup[u.Path] = true
						if _, err := ctx.Q.SendStringGet(urlStr); err != nil {
							fmt.Errorf("Error queuing: %s\n", err)
						} else {
							//fmt.Printf("Found internal link: %s\n", urlStr)
						}
					}
					mu.Unlock()
				}
			})
		}))

	f := fetchbot.New(mux)

	queue := f.Start()
	queue.SendStringGet(rootUrlStr)
	queue.Block()
	fmt.Printf("Done\n")
}
