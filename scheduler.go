package main

import (
	"log"
	"time"
)

func startScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			refreshDueFeeds()
		}
	}()
}

func refreshDueFeeds() {
	feeds, err := FeedsDueForRefresh()
	if err != nil {
		log.Printf("scheduler: failed to query due feeds: %v", err)
		return
	}

	for _, f := range feeds {
		regexes := SelectorRegexes{
			Title:       f.TitleRegex,
			Link:        f.LinkRegex,
			Description: f.DescriptionRegex,
			PubDate:     f.PubDateRegex,
		}
		items, err := FetchAndParse(f.URL, f.ItemSelector, f.TitleSelector, f.LinkSelector, f.DescriptionSelector, f.PubDateSelector, regexes)
		if err != nil {
			log.Printf("scheduler: failed to fetch feed %d (%s): %v", f.ID, f.Name, err)
			continue
		}

		rssXML, err := GenerateRSS(f.Name, f.URL, items)
		if err != nil {
			log.Printf("scheduler: failed to generate RSS for feed %d (%s): %v", f.ID, f.Name, err)
			continue
		}

		err = UpdateFeedCache(f.ID, rssXML)
		if err != nil {
			log.Printf("scheduler: failed to update cache for feed %d (%s): %v", f.ID, f.Name, err)
			continue
		}

		log.Printf("scheduler: refreshed feed %d (%s), %d items", f.ID, f.Name, len(items))
	}
}
