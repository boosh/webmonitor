# Web monitor
A simple go desktop app to poll a web page and notify you when it changes.

## Usage
1. Clone the repo
2. Compile with `go build`
3. Run it passing the url as an argument: `./web-monitor https://www.example.com`
4. The app will poll the page every `DELAY` seconds (see code) and pop up an alert when it changes.
