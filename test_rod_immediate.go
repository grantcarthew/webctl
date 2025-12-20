package main

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

func main() {
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	fmt.Println("Test 1: MustPage with URL, then IMMEDIATE HTML...")
	start1 := time.Now()
	page1 := browser.MustPage("https://example.com/")
	pageTime1 := time.Since(start1)

	// NO DELAY - immediate HTML call
	htmlStart1 := time.Now()
	html1 := page1.MustHTML()
	htmlTime1 := time.Since(htmlStart1)

	total1 := time.Since(start1)
	fmt.Printf("  MustPage: %v\n", pageTime1)
	fmt.Printf("  MustHTML: %v\n", htmlTime1)
	fmt.Printf("  Total: %v\n", total1)
	fmt.Printf("  HTML length: %d\n\n", len(html1))

	// Test 2: Separate Navigate and HTML
	fmt.Println("Test 2: MustPage + MustNavigate + IMMEDIATE HTML...")
	start2 := time.Now()
	page2 := browser.MustPage()
	createTime2 := time.Since(start2)

	navStart2 := time.Now()
	page2.MustNavigate("https://example.com/")
	navTime2 := time.Since(navStart2)

	// NO DELAY - immediate HTML call
	htmlStart2 := time.Now()
	html2 := page2.MustHTML()
	htmlTime2 := time.Since(htmlStart2)

	total2 := time.Since(start2)
	fmt.Printf("  MustPage (no URL): %v\n", createTime2)
	fmt.Printf("  MustNavigate: %v\n", navTime2)
	fmt.Printf("  MustHTML: %v\n", htmlTime2)
	fmt.Printf("  Total: %v\n", total2)
	fmt.Printf("  HTML length: %d\n", len(html2))
}
