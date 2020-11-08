package main

import (
	"flag"
	"fmt"
	"log"

	"./ucl"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	// Retrieve command line arguments
	var maxCalls int
	var maxDepth int
	flag.IntVar(&maxCalls, "p", 1, "Specify parallel calls. Default is 1")
	flag.IntVar(&maxDepth, "d", 1, "Specify max depth you want to crawl. Default is 1")
	flag.Parse()

	totalArticles := 0
	maxErrors := 5
	currentErrors := 0

	// calledURLs is used to make sure we don't call the same url twice
	calledURLs := map[string]bool{}

	// Initialize the caller
	caller := ucl.URLCaller{
		MaxCalls: maxCalls,
	}
	// Call the base URL
	baseURL := "https://www.breakit.se"
	caller.Get(&ucl.PageRequest{
		URL:   baseURL,
		Depth: 0,
	})

	calledURLs[baseURL] = true

	handleResponse := func() bool {
		// Retrieve the next completed response
		response := caller.Next()
		// If the response is nil that means there are no more calls in the pipeline
		if response == nil {
			return true
		}

		// If the call returned an error we break
		if response.Err != nil {
			log.Fatal(response.Err)
		}
		defer response.Response.Body.Close()
		// If the status error is 4xx we are responsible somehow and we end the program
		if response.Response.StatusCode >= 400 && response.Response.StatusCode < 500 {
			log.Fatalf("Error StatusCdoe: %d %s", response.Response.StatusCode, response.Response.Status)
		}

		// If the statuscode is not 200 it means the server had an error, this can happen so we try to requeue the job and hope it succeeds.
		// When we get to many errors we terminate the program
		if response.Response.StatusCode != 200 {
			currentErrors++
			if currentErrors >= maxErrors {
				log.Fatal("Server returned to many errors")
			}

			caller.Get(response.Request)
		}

		// Create a goquery document
		doc, err := goquery.NewDocumentFromReader(response.Response.Body)
		if err != nil {
			log.Fatal(response.Err)
		}

		// Find all links on the page
		if response.Request.Depth < maxDepth {
			doc.Find("a").Each(func(i int, s *goquery.Selection) {
				href, ok := s.Attr("href")
				// Check for valid link
				if !ok {
					return
				}
				if href == "" {
					return
				}

				// Remove self links
				if href == "/" {
					return
				}

				// Make Sure we don't leave the page
				if href[0] != '/' {
					return
				}

				// Check if we called this URL before, if we did we skip adding it
				url := baseURL + href
				if _, ok := calledURLs[url]; ok {
					return
				}
				calledURLs[url] = true

				// Add links to the queue
				caller.Get(&ucl.PageRequest{
					URL:   url,
					Depth: response.Request.Depth + 1,
				})
			})
		}

		// Check if it's an article page if it isn't we move on
		if !doc.Find("html").HasClass("articlePage") {
			return false
		}

		// Print out the information in the article
		fmt.Println("URL: " + response.Request.URL)
		fmt.Println("Date: " + doc.Find(".article__date").Text())
		fmt.Println("H1: " + doc.Find(".article__title").Text())
		fmt.Println("H4: " + doc.Find(".article__preamble").Text())
		fmt.Println("Paragraph: " + doc.Find(".article__body").Find("p").Text())
		totalArticles++
		return false
	}

	// Do handle response until the function says we are done
	for !handleResponse() {
	}

	fmt.Println("")
	fmt.Printf("Done with crawling, crawled %d articles", totalArticles)
}
