package main

import (
	"flag"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

func main() {
	log.SetLevel(log.InfoLevel)

	var textExists string
	var delay uint

	// Define flag arguments
	flag.StringVar(&textExists, "text-exists", "", "Alert only if this text exists on the page")
	flag.UintVar(&delay, "delay", 30, "Poll interval in seconds")

	flag.Usage = usage

	// Parse command-line flags
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		log.Fatal("Please provide a URL to check and MP3 file to play")
		usage()
		os.Exit(1)
	}

	urlStr := args[0]
	mp3Path := args[1]

	fmt.Printf("Will check URL "+urlStr+" for changes every %ds\n", delay)

	myApp := app.New()
	myApp.Settings().SetTheme(theme.LightTheme())
	myWindow := myApp.NewWindow("Web Monitor")

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.Fatal(err)
	}

	link := widget.NewHyperlink("Checking "+urlStr+" for changes...", parsedURL)
	statusLabel := widget.NewLabel("")
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	content := container.NewVBox(
		link,
		statusLabel,
	)

	myWindow.SetContent(content)

	myWindow.Resize(fyne.NewSize(300, 200))

	go pollForChanges(urlStr, mp3Path, delay, textExists, myWindow, statusLabel)

	myWindow.ShowAndRun()
}

func usage() {
	fmt.Printf("Usage: %s [flags] <URL> <MP3 file>\n", os.Args[0])
	fmt.Println("Poll a web page for changes and play the MP3 file when a change is detected")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

// Poll a web page for changes and play an MP3 file when a change is detected
func pollForChanges(urlStr string, mp3Path string, delay uint, textExists string, myWindow fyne.Window, statusLabel *widget.Label) {
	var originalText string
	var pageChanged bool

	for {
		if originalText == "" {
			originalText = getPage(urlStr)
			log.Debugf("Original text: %s", originalText)
			statusLabel.SetText(fmt.Sprintf("[%s] Initial page retrieved", time.Now().Format("2006-01-02 15:04:05")))
			continue
		}

		time.Sleep(time.Duration(delay) * time.Second)

		pageText := getPage(urlStr)

		if pageText == "" {
			log.Warn("Empty page text")
			statusLabel.SetText(fmt.Sprintf("[%s] Empty page text", time.Now().Format("2006-01-02 15:04:05")))
			continue
		}

		log.Debugf("Current text: %s", pageText)

		pageChanged = false

		if textExists != "" {
			pageChanged = strings.Contains(strings.ToLower(pageText), strings.ToLower(textExists))
		} else {
			pageChanged = pageText != originalText
		}

		if pageChanged {
			log.Info("Web page change detected")

			statusLabel.SetText(fmt.Sprintf("[%s] Page has changed", time.Now().Format("2006-01-02 15:04:05")))

			go func() {
				showAlert(myWindow, urlStr)
			}()

			if mp3Path != "" {
				log.Info("Playing sound")
				playMp3(mp3Path)
			}

			originalText = pageText
		} else {
			log.Infof("[%s] Page unchanged", time.Now().Format("2006-01-02 15:04:05"))
			statusLabel.SetText(fmt.Sprintf("[%s] Page unchanged", time.Now().Format("2006-01-02 15:04:05")))
		}
	}
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
		log.Error(err)
		return ""
	}

	return pageText.String()
}

func showAlert(window fyne.Window, url string) {
	alert := dialog.NewInformation("Alert", "Web page has changed: "+url, window)
	alert.Show()

	window.Hide()
	window.Show()
}
