package main

import (
	"fmt"
	"time"

	"github.com/gorilla/feeds"
)

func GenerateRSS(feedName, feedURL string, items []FeedItem) (string, error) {
	now := time.Now()
	feed := &feeds.Feed{
		Title:   feedName,
		Link:    &feeds.Link{Href: feedURL},
		Updated: now,
	}

	for _, item := range items {
		created := now
		if item.PubDate != "" {
			if parsed, err := parseDate(item.PubDate); err == nil {
				created = parsed
			}
		}
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       item.Title,
			Link:        &feeds.Link{Href: item.Link},
			Description: item.Description,
			Created:     created,
		})
	}

	return feed.ToRss()
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		"2006-01-02",
		"January 2, 2006",
		"Jan 2, 2006",
		"02 Jan 2006",
		"2006/01/02",
		"01/02/2006",
		"02-01-2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}
