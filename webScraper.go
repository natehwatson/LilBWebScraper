package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
)

var writingErrors []error

func writeToFile(title string, lyrics string) {
	directory := "./lyrics"
	filename := fmt.Sprintf("%s.txt", title)
	fullpath := directory + "/" + filename

	fileContents := fmt.Sprintf("%s\n\n%s", title, lyrics)
	data := []byte(fileContents)
	err := os.WriteFile(fullpath, data, 0777) // 0777 means read/write/execute for all
	if err != nil {
		fmt.Println(err)
		writingErrors = append(writingErrors, err)
	}
	fmt.Println("Wrote to file:", filename)
}

func endScrape() {
	if len(writingErrors) > 0 {
		fmt.Println("Finished scraping with errors:")
		for i := 0; i < len(writingErrors); i++ {
			fmt.Println(writingErrors[i])
		}
	}
}

func main() {
	numVisited := 0
	var url string = "https://genius.com/artists/Lil-b/albums"
	c := colly.NewCollector(
		colly.AllowedDomains("genius.com"),
		colly.AllowURLRevisit(),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 14.4; rv:124.0) Gecko/20100101 Firefox/124.0"),
		colly.Debugger(&debug.LogDebugger{}),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob: "*genius.*",
		Delay:      1 * time.Second,
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting: ", r.URL.String())
		numVisited++
		fmt.Println("num visited:", numVisited)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Response received")
	})

	// scrape discog for album links
	c.OnHTML("ul.ListSection-desktop__Items-sc-f0fb85c4-8", func(e *colly.HTMLElement) {
		fmt.Println("ul found")
		e.ForEach("li", func(i int, ee *colly.HTMLElement) {
			link, _ := ee.DOM.Find("a[href]").Attr("href")
			if strings.Contains(link, "https://genius.com/albums/Lil-b") {
				ee.Request.Visit(link)
			}
		})
	})

	// scrape album page for song links
	c.OnHTML("div.chart_row-content", func(e *colly.HTMLElement) {
		songLink, _ := e.DOM.Find("a[href]").Attr("href")
		e.Request.Visit(songLink)
	})

	// scrape lyrics
	c.OnHTML("div#lyrics-root", func(e *colly.HTMLElement) {
		e.ForEach("div", func(i int, ee *colly.HTMLElement) {
			if strings.EqualFold(ee.Attr("data-lyrics-container"), "true") {
				lyricsContainer := ee.DOM.Clone()          // get lyrics container
				title := lyricsContainer.Find("h2").Text() // get song title

				//clean lyrics and title
				title = strings.Replace(title, " Lyrics", "", 1) // remove " Lyrics" suffix
				lyricsContainer.Find("div").Remove()             // remove div containing excess info
				lyricsContainer.Find("br").ReplaceWithHtml("\n") // replace <br> with newlines
				lyricsText := lyricsContainer.Text()             // get cleaned lyrics text

				// save lyrics to file
				writeToFile(title, lyricsText)
			}
		})
	})

	// order of callbacks
	// c.OnRequest(func(r *colly.Request) {
	// c.OnResponse
	// c.OnHtml
	// c.OnScraped

	c.Visit(url)
	defer endScrape()
}

// idea:
// 1. accumulate a queue of links to songs
// 2. for each link, visit the page and scrape the lyrics

// path:
// from discog page -> ul -> li -> a[href] -> visit each href to album page
// from album page ->
