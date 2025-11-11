package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
)

var writingErrors []error
var currentAlbum string
var currentReleaseDate string
var trackNumber int
var songTitle string
var structList []APIResponse

type APIResponse struct {
	Response Response `json:"response"`
}

type Response struct {
	Albums   []Album `json:"albums"`
	NextPage int     `json:"next_page"` // pointer because it can be null / missing
}

type Album struct {
	Name                  string      `json:"name"`
	URL                   string      `json:"url"`
	ReleaseDateComponents ReleaseDate `json:"release_date_components"`
}

type ReleaseDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

// save lyrics to local files
func writeToFile(title string, lyrics string) {
	directory := "./lyrics/" + currentAlbum + " (" + currentReleaseDate + ")"
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

func getJson(url string) APIResponse {

	response, err := http.Get(url)
	if err != nil {
		fmt.Printf("error fetching JSON data: %s", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Println("recieved not-OK status: ", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("error reading response body: %v", err)
	}

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		fmt.Printf("error unmarshaling JSON: %v", err)
	}

	return apiResp
}

// retrieve all album URLs from paginated JSON data
func getJSONStructs(url string) {

	// separate url into pre and post page components
	prePageURL, postPageURL, found := strings.Cut(url, "?page=")
	AmpersandIndex := strings.Index(postPageURL, "&")
	if !found {

	}
	postPageURL = postPageURL[AmpersandIndex:]

	//get data
	isMorePages := true
	i := 1
	for isMorePages {
		JSONurl := prePageURL + "?page=" + fmt.Sprint(i) + postPageURL[AmpersandIndex:]
		responseStruct := getJson(JSONurl)

		structList = append(structList, responseStruct)

		nextPage := responseStruct.Response.NextPage
		if nextPage == 0 {
			isMorePages = false
		}
		i++
	}
}

func main() {
	url := "https://genius.com/api/artists/455/albums?page=1&per_page=50&sort=release_date&pageviews=false&text_format=html%2Cmarkdown"
	getJSONStructs(url)

	numVisited := 0
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

	for i := 0; i < len(structList); i++ {
		for j := 0; j < len(structList[i].Response.Albums); j++ {
			trackNumber = 0
			currentAlbum = structList[i].Response.Albums[j].Name
			currentReleaseDate = fmt.Sprintf("%04d-%02d-%02d", structList[i].Response.Albums[j].ReleaseDateComponents.Year, structList[i].Response.Albums[j].ReleaseDateComponents.Month, structList[i].Response.Albums[j].ReleaseDateComponents.Day)
			c.Visit(structList[i].Response.Albums[j].URL)
		}
	}
	defer endScrape()
}
