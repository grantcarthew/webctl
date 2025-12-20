package main

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

func main() {
	// Launch browser with slow motion to see timing
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	fmt.Println("Creating page...")
	page := browser.MustPage()

	fmt.Println("Starting navigation to example.com...")
	navStart := time.Now()

	// Navigate - this returns when navigation completes
	page.MustNavigate("https://example.com/")

	navElapsed := time.Since(navStart)
	fmt.Printf("Navigation completed in: %v\n", navElapsed)

	// Small delay to see if Rod waits for anything
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\nGetting HTML...")
	htmlStart := time.Now()

	// Get HTML
	html := page.MustHTML()

	htmlElapsed := time.Since(htmlStart)
	fmt.Printf("HTML retrieved in: %v\n", htmlElapsed)
	fmt.Printf("Total time (nav + html): %v\n", time.Since(navStart))
	fmt.Printf("HTML length: %d bytes\n", len(html))

	// Also test getting it again to see if there's caching
	fmt.Println("\nGetting HTML again...")
	html2Start := time.Now()
	html2 := page.MustHTML()
	fmt.Printf("Second HTML retrieval: %v\n", time.Since(html2Start))
	fmt.Printf("Same HTML: %v\n", html == html2)
}
