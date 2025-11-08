package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
)

var writingErrors []error
var currentAlbum string
var trackNumber int
var songTitle string

// save lyrics to local files
func writeToFile(title string, lyrics string) {
	directory := "./lyrics/" + currentAlbum
	// create directory if it doesn't exist
	err := os.MkdirAll(directory, 0777)
	if err != nil {
		writingErrors = append(writingErrors, err)
	}
	filename := fmt.Sprintf("%s %s.txt", strconv.Itoa(trackNumber), title)
	fullpath := directory + "/" + filename

	fileContents := fmt.Sprintf("%s\n\n%s", title, lyrics)
	data := []byte(fileContents)
	err = os.WriteFile(fullpath, data, 0777) // 0777 means read/write/execute for all
	if err != nil {
		fmt.Println(err)
		writingErrors = append(writingErrors, err)
	}
	fmt.Println("Wrote to file:", filename)
}

func trimTitle(songTitle string) string {
	songTitle = strings.ReplaceAll(songTitle, "\n", "")
	songTitle = strings.ReplaceAll(songTitle, "Lyrics", "")
	songTitle = strings.TrimSpace(songTitle)
	return songTitle
}

// print if there were any errors during scraping
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
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 14.4; rv:124.0) Gecko/20100101 Firefox/124.0"),
		colly.Debugger(&debug.LogDebugger{}),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob: "*genius.*",
		Delay:      1 * time.Second,
	})

	c.OnError(func(r *colly.Response, err error) {

		writeToFile(songTitle, "Missing lyrics")
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
				currentAlbum = ee.DOM.Find("h3").Text()
				trackNumber = 0
				ee.Request.Visit(link)
			}
		})
	})

	// scrape album page for song links
	c.OnHTML("div.chart_row-content", func(e *colly.HTMLElement) {
		trackNumber += 1
		songLink, _ := e.DOM.Find("a[href]").Attr("href")
		songTitle = e.DOM.Find("a[href]").Text()
		songTitle = trimTitle(songTitle)
		fmt.Println("Visiting song:", songTitle+" (Track "+strconv.Itoa(trackNumber)+")")
		e.Request.Visit(songLink)
	})

	// scrape lyrics
	c.OnHTML("div#lyrics-root", func(e *colly.HTMLElement) {
		var lyricsString string
		e.ForEach("div", func(i int, ee *colly.HTMLElement) {
			if strings.EqualFold(ee.Attr("data-lyrics-container"), "true") {
				// get lyrics container
				lyricsContainer := ee.DOM.Clone()

				// get song title and remove lyrics suffix for first container only
				//clean lyrics and title
				lyricsContainer.Find("div").Remove()             // remove div containing excess info
				lyricsContainer.Find("br").ReplaceWithHtml("\n") // replace <br> with newlines
				lyricsText := lyricsContainer.Text()             // get cleaned lyrics text
				lyricsString += lyricsText

			}
		})

		// save lyrics to file
		writeToFile(songTitle, lyricsString)
	})

	c.Visit(url)
	defer endScrape()
}
