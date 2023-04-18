package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

const DELAY = 5 // delay between checks in seconds

func main() {
	log.SetLevel(log.InfoLevel)

	if len(os.Args) < 2 {
		log.Fatal("Please provide a URL to check")
		usage()
		os.Exit(1)
	}

	url := os.Args[1]

	mp3Path := ""
	if len(os.Args) == 3 {
		mp3Path = os.Args[2]
	}

	fmt.Printf("Will check URL "+url+" for changes every %ds\n", DELAY)

	myApp := app.New()
	myApp.Settings().SetTheme(theme.LightTheme())
	myWindow := myApp.NewWindow("Web Monitor")

	myWindow.Resize(fyne.NewSize(300, 200))

	var originalText string

	go func() {
		for {
			if originalText == "" {
				originalText = getPage(url)
				log.Debugf("Original text: %s", originalText)
				continue
			}

			time.Sleep(DELAY * time.Second)

			pageText := getPage(url)
			log.Debugf("Current text: %s", pageText)

			if pageText != originalText {
				log.Info("Web page change detected")

				go func() {
					showAlert(myWindow, url)
				}()

				if mp3Path != "" {
					log.Info("Playing sound")
					playMp3(mp3Path)
				}

				myApp.Quit()
			} else {
				log.Infof("[%s] Page unchanged", time.Now().Format("2006-01-02 15:04:05"))
			}
		}
	}()

	myWindow.ShowAndRun()
}

func usage() {
	fmt.Println("Usage: web-monitor <URL> [MP3 file]")
}

// Play an MP3 file
func playMp3(mp3Path string) {
	// Open the sound file
	file, err := os.Open(mp3Path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Decode the MP3 file
	streamer, format, err := mp3.Decode(file)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()

	// Initialize the speaker with the sample rate of the MP3 file
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Fatal(err)
	}

	// Play the MP3 file and wait for it to finish
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		// Close the speaker and exit when the file is finished playing
		speaker.Close()
		os.Exit(0)
	})))

	// Keep the program running until the audio finishes playing
	select {}
}

func getPage(url string) string {
	// Create a new collector
	c := colly.NewCollector()

	// Set User-Agent to simulate a browser request
	extensions.RandomUserAgent(c)

	// Variable to store the extracted text content
	var pageText strings.Builder

	// Callback for when visiting a HTML element
	c.OnHTML("*", func(e *colly.HTMLElement) {
		text := strings.TrimSpace(e.Text)
		if text != "" {
			pageText.WriteString(text)
			pageText.WriteString("\n")
		}
	})

	// Callback for error handling
	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	// Start the web scraping by visiting the target URL
	err := c.Visit(url)
	if err != nil {
		log.Fatal(err)
	}

	return pageText.String()
}

func showAlert(window fyne.Window, url string) {
	alert := dialog.NewInformation("Alert", "Web page has changed: "+url, window)
	alert.Show()

	window.Hide()
	window.Show()
}
