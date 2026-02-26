package main

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Feed struct {
	ID                  int64
	Name                string
	URL                 string
	FeedType            string // "scrape" or "rss"
	RSSURL              string
	ItemSelector        string
	TitleSelector       string
	LinkSelector        string
	DescriptionSelector string
	PubDateSelector     string
	TitleRegex          string
	LinkRegex           string
	DescriptionRegex    string
	PubDateRegex        string
	RefreshMinutes      int
	CachedRSS          string
	LastRefreshed       *time.Time
	CreatedAt           time.Time
}

var db *sql.DB

func initDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		feed_type TEXT NOT NULL DEFAULT 'scrape',
		rss_url TEXT NOT NULL DEFAULT '',
		item_selector TEXT NOT NULL,
		title_selector TEXT NOT NULL,
		link_selector TEXT NOT NULL,
		description_selector TEXT NOT NULL,
		pub_date_selector TEXT NOT NULL DEFAULT '',
		title_regex TEXT NOT NULL DEFAULT '',
		link_regex TEXT NOT NULL DEFAULT '',
		description_regex TEXT NOT NULL DEFAULT '',
		pub_date_regex TEXT NOT NULL DEFAULT '',
		refresh_minutes INTEGER NOT NULL DEFAULT 60,
		cached_rss TEXT NOT NULL DEFAULT '',
		last_refreshed DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}

	// Migrate: add columns to existing tables that lack them
	db.Exec(`ALTER TABLE feeds ADD COLUMN pub_date_selector TEXT NOT NULL DEFAULT ''`)
	db.Exec(`ALTER TABLE feeds ADD COLUMN title_regex TEXT NOT NULL DEFAULT ''`)
	db.Exec(`ALTER TABLE feeds ADD COLUMN link_regex TEXT NOT NULL DEFAULT ''`)
	db.Exec(`ALTER TABLE feeds ADD COLUMN description_regex TEXT NOT NULL DEFAULT ''`)
	db.Exec(`ALTER TABLE feeds ADD COLUMN pub_date_regex TEXT NOT NULL DEFAULT ''`)
	db.Exec(`ALTER TABLE feeds ADD COLUMN feed_type TEXT NOT NULL DEFAULT 'scrape'`)
	db.Exec(`ALTER TABLE feeds ADD COLUMN rss_url TEXT NOT NULL DEFAULT ''`)

	return nil
}

func CreateFeed(f *Feed) (int64, error) {
	if f.FeedType == "" {
		f.FeedType = "scrape"
	}
	result, err := db.Exec(
		`INSERT INTO feeds (name, url, feed_type, rss_url, item_selector, title_selector, link_selector, description_selector, pub_date_selector, title_regex, link_regex, description_regex, pub_date_regex, refresh_minutes, cached_rss, last_refreshed)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.Name, f.URL, f.FeedType, f.RSSURL, f.ItemSelector, f.TitleSelector, f.LinkSelector, f.DescriptionSelector,
		f.PubDateSelector, f.TitleRegex, f.LinkRegex, f.DescriptionRegex, f.PubDateRegex,
		f.RefreshMinutes, f.CachedRSS, f.LastRefreshed,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func GetFeed(id int64) (*Feed, error) {
	f := &Feed{}
	err := db.QueryRow(
		`SELECT id, name, url, feed_type, rss_url, item_selector, title_selector, link_selector, description_selector,
		        pub_date_selector, title_regex, link_regex, description_regex, pub_date_regex,
		        refresh_minutes, cached_rss, last_refreshed, created_at
		 FROM feeds WHERE id = ?`, id,
	).Scan(&f.ID, &f.Name, &f.URL, &f.FeedType, &f.RSSURL, &f.ItemSelector, &f.TitleSelector, &f.LinkSelector,
		&f.DescriptionSelector, &f.PubDateSelector, &f.TitleRegex, &f.LinkRegex, &f.DescriptionRegex, &f.PubDateRegex,
		&f.RefreshMinutes, &f.CachedRSS, &f.LastRefreshed, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func ListFeeds() ([]Feed, error) {
	rows, err := db.Query(
		`SELECT id, name, url, feed_type, rss_url, item_selector, title_selector, link_selector, description_selector,
		        pub_date_selector, title_regex, link_regex, description_regex, pub_date_regex,
		        refresh_minutes, cached_rss, last_refreshed, created_at
		 FROM feeds ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []Feed
	for rows.Next() {
		var f Feed
		err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.FeedType, &f.RSSURL, &f.ItemSelector, &f.TitleSelector, &f.LinkSelector,
			&f.DescriptionSelector, &f.PubDateSelector, &f.TitleRegex, &f.LinkRegex, &f.DescriptionRegex, &f.PubDateRegex,
		&f.RefreshMinutes, &f.CachedRSS, &f.LastRefreshed, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}

func DeleteFeed(id int64) error {
	_, err := db.Exec(`DELETE FROM feeds WHERE id = ?`, id)
	return err
}

func UpdateFeed(f *Feed) error {
	_, err := db.Exec(
		`UPDATE feeds SET name = ?, url = ?, rss_url = ?, item_selector = ?, title_selector = ?, link_selector = ?,
		 description_selector = ?, pub_date_selector = ?, title_regex = ?, link_regex = ?,
		 description_regex = ?, pub_date_regex = ?, refresh_minutes = ?
		 WHERE id = ?`,
		f.Name, f.URL, f.RSSURL, f.ItemSelector, f.TitleSelector, f.LinkSelector,
		f.DescriptionSelector, f.PubDateSelector, f.TitleRegex, f.LinkRegex,
		f.DescriptionRegex, f.PubDateRegex, f.RefreshMinutes, f.ID,
	)
	return err
}

func UpdateFeedCache(id int64, rssXML string) error {
	_, err := db.Exec(
		`UPDATE feeds SET cached_rss = ?, last_refreshed = ? WHERE id = ?`,
		rssXML, time.Now().UTC().Format("2006-01-02 15:04:05"), id,
	)
	return err
}

func FeedsDueForRefresh() ([]Feed, error) {
	rows, err := db.Query(
		`SELECT id, name, url, feed_type, rss_url, item_selector, title_selector, link_selector, description_selector,
		        pub_date_selector, title_regex, link_regex, description_regex, pub_date_regex,
		        refresh_minutes, cached_rss, last_refreshed, created_at
		 FROM feeds
		 WHERE feed_type = 'scrape'
		   AND (last_refreshed IS NULL
		    OR datetime(last_refreshed, '+' || refresh_minutes || ' minutes') <= datetime('now'))`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []Feed
	for rows.Next() {
		var f Feed
		err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.FeedType, &f.RSSURL, &f.ItemSelector, &f.TitleSelector, &f.LinkSelector,
			&f.DescriptionSelector, &f.PubDateSelector, &f.TitleRegex, &f.LinkRegex, &f.DescriptionRegex, &f.PubDateRegex,
		&f.RefreshMinutes, &f.CachedRSS, &f.LastRefreshed, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}
