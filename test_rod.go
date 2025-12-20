package main

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

func main() {
	// Launch browser
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	// Navigate to example.com
	page := browser.MustPage("https://example.com/")

	fmt.Println("Navigation complete, getting HTML...")
	start := time.Now()

	// Get HTML
	html := page.MustHTML()

	elapsed := time.Since(start)
	fmt.Printf("HTML retrieved in: %v\n", elapsed)
	fmt.Printf("HTML length: %d bytes\n", len(html))
}
