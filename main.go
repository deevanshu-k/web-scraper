package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"

	"golang.org/x/net/html"
)

func scrapHtmlBody(n *html.Node, baseUrl *string, linksChannel chan<- string) {
	// If Node is of type <a></a> tag
	if n.Type == html.ElementNode && n.Data == "a" {
		// Iterate over attributes
		for _, attr := range n.Attr {
			// If attribute is 'href'
			if attr.Key == "href" && len(attr.Val) > 0 {
				// If link not have baseURL add it
				// Send link via channel
				if attr.Val[0] == '/' {
					linksChannel <- *baseUrl + attr.Val
				} else {
					linksChannel <- attr.Val
				}
			}
		}
	}

	// Recursively call other siblings
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		scrapHtmlBody(c, baseUrl, linksChannel)
	}
}

func scrapLink(link string, baseUrl *string, wg *sync.WaitGroup, linksChannel chan<- string, linkMap map[string]bool) error {
	defer wg.Done()
	fmt.Println(color.GreenString("Scraping Link: "), link)

	// Do request
	resp, err := http.Get(link)
	if err != nil {
		fmt.Println(color.RedString("Dead Link: "), link)
		linkMap[link] = false
		return err
	}
	defer resp.Body.Close()

	// If custom server errors
	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		fmt.Println(color.RedString("Dead Link: "), link, resp.Status)
		linkMap[link] = false
	}

	// If link is of 3rd party
	if !strings.Contains(link, *baseUrl) {
		return nil
	}

	// Parse HTML Body
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return err
	}

	// Iterative over HTML Node
	scrapHtmlBody(doc, baseUrl, linksChannel)

	return nil
}

func main() {
	startTime := time.Now()

	// Get url from flag
	urlFlagData := flag.String("url", "", "pass valid url")
	flag.Parse()
	if *urlFlagData == "" {
		log.Fatal("Need url !")
	}

	// Variables
	var wg sync.WaitGroup
	var wgmain sync.WaitGroup
	var linkMap = make(map[string]bool)
	linksChannel := make(chan string, 1)

	// Add the first URL to start scraping
	linksChannel <- *urlFlagData

	// Close channel after scrapping is complete
	wgmain.Add(1)
	go func() {
		time.Sleep(time.Second)
		wg.Wait()
		close(linksChannel)
		wgmain.Done()
	}()

	// Get links from links channel
	for link := range linksChannel {
		if _, ok := linkMap[link]; !ok {
			linkMap[link] = true
			wg.Add(1)
			go scrapLink(link, urlFlagData, &wg, linksChannel, linkMap)
		}
	}

	// Result
	wgmain.Wait()
	fmt.Println("\n+----- " + color.RedString(" Dead Links ") + "------")
	for link, isAccessible := range linkMap {
		if !isAccessible {
			fmt.Println(color.YellowString(link))
		}
	}
	fmt.Println(color.BlueString("Total Time: "), time.Since(startTime))
}
