package main

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type SelectorRegexes struct {
	Title       string
	Link        string
	Description string
	PubDate     string
}

type FeedItem struct {
	Title       string
	Link        string
	Description string
	PubDate     string
}

func FetchAndParse(sourceURL, itemSel, titleSel, linkSel, descSel, pubDateSel string, regexes SelectorRegexes) ([]FeedItem, error) {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: status %d", sourceURL, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	return ExtractItems(doc, sourceURL, itemSel, titleSel, linkSel, descSel, pubDateSel, regexes), nil
}

func ExtractItems(doc *goquery.Document, sourceURL, itemSel, titleSel, linkSel, descSel, pubDateSel string, regexes SelectorRegexes) []FeedItem {
	var items []FeedItem

	baseURL, _ := url.Parse(sourceURL)

	doc.Find(itemSel).Each(func(i int, s *goquery.Selection) {
		item := FeedItem{}

		if titleSel != "" {
			item.Title = applyRegex(strings.TrimSpace(s.Find(titleSel).First().Text()), regexes.Title)
		}

		if linkSel != "" {
			linkEl := s.Find(linkSel).First()
			if href, exists := linkEl.Attr("href"); exists {
				item.Link = resolveURL(baseURL, applyRegex(strings.TrimSpace(href), regexes.Link))
			}
		}

		if descSel != "" {
			item.Description = applyRegex(strings.TrimSpace(s.Find(descSel).First().Text()), regexes.Description)
		}

		if pubDateSel != "" {
			item.PubDate = applyRegex(strings.TrimSpace(s.Find(pubDateSel).First().Text()), regexes.PubDate)
		}

		if item.Title != "" || item.Link != "" {
			items = append(items, item)
		}
	})

	return items
}

func applyRegex(text, pattern string) string {
	if pattern == "" {
		return text
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return text
	}
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 2 {
		return matches[1]
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return text
}

func resolveURL(base *url.URL, rawURL string) string {
	if base == nil {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return base.ResolveReference(parsed).String()
}
