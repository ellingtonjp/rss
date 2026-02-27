package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

var pageTmpl map[string]*template.Template

func initTemplates() {
	layout := "templates/layout.html"
	pageTmpl = map[string]*template.Template{
		"index":           template.Must(template.ParseFS(staticFiles, layout, "templates/index.html")),
		"preview":         template.Must(template.ParseFS(staticFiles, layout, "templates/preview.html")),
		"feeds":           template.Must(template.ParseFS(staticFiles, layout, "templates/feeds.html")),
		"feed_detail":     template.Must(template.ParseFS(staticFiles, layout, "templates/feed_detail.html")),
		"feed_edit":       template.Must(template.ParseFS(staticFiles, layout, "templates/feed_edit.html")),
		"feed_edit_rss":   template.Must(template.ParseFS(staticFiles, layout, "templates/feed_edit_rss.html")),
		"preview_results": template.Must(template.ParseFS(staticFiles, "templates/preview_results.html")),
	}
}

func renderPage(w http.ResponseWriter, page string, data any) {
	tmpl, ok := pageTmpl[page]
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// For partial templates (no layout), execute by template name
	if page == "preview_results" {
		err := tmpl.Execute(w, data)
		if err != nil {
			log.Printf("template error: %v", err)
		}
		return
	}
	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "index", nil)
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	sourceURL := r.FormValue("url")
	if sourceURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	data := struct{ URL string }{URL: sourceURL}
	renderPage(w, "preview", data)
}

func handlePreviewTest(w http.ResponseWriter, r *http.Request) {
	sourceURL := r.FormValue("url")
	itemSel := r.FormValue("item_selector")
	titleSel := r.FormValue("title_selector")
	linkSel := r.FormValue("link_selector")
	descSel := r.FormValue("description_selector")
	pubDateSel := r.FormValue("pub_date_selector")
	regexes := SelectorRegexes{
		Title:       r.FormValue("title_regex"),
		Link:        r.FormValue("link_regex"),
		Description: r.FormValue("description_regex"),
		PubDate:     r.FormValue("pub_date_regex"),
	}

	if sourceURL == "" || itemSel == "" {
		http.Error(w, "URL and item selector are required", http.StatusBadRequest)
		return
	}

	items, err := FetchAndParse(sourceURL, itemSel, titleSel, linkSel, descSel, pubDateSel, regexes)
	if err != nil {
		fmt.Fprintf(w, `<p class="error">Error: %s</p>`, template.HTMLEscapeString(err.Error()))
		return
	}

	data := struct {
		Items []FeedItem
		Count int
	}{Items: items, Count: len(items)}

	renderPage(w, "preview_results", data)
}

func handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	refreshMinutes, _ := strconv.Atoi(r.FormValue("refresh_minutes"))
	if refreshMinutes < 1 {
		refreshMinutes = 60
	}

	f := &Feed{
		Name:                r.FormValue("name"),
		URL:                 r.FormValue("url"),
		ItemSelector:        r.FormValue("item_selector"),
		TitleSelector:       r.FormValue("title_selector"),
		LinkSelector:        r.FormValue("link_selector"),
		DescriptionSelector: r.FormValue("description_selector"),
		PubDateSelector:     r.FormValue("pub_date_selector"),
		TitleRegex:          r.FormValue("title_regex"),
		LinkRegex:           r.FormValue("link_regex"),
		DescriptionRegex:    r.FormValue("description_regex"),
		PubDateRegex:        r.FormValue("pub_date_regex"),
		RefreshMinutes:      refreshMinutes,
	}

	if f.Name == "" || f.URL == "" || f.ItemSelector == "" {
		http.Error(w, "Name, URL, and item selector are required", http.StatusBadRequest)
		return
	}

	// Generate initial RSS
	regexes := SelectorRegexes{
		Title:       f.TitleRegex,
		Link:        f.LinkRegex,
		Description: f.DescriptionRegex,
		PubDate:     f.PubDateRegex,
	}
	items, err := FetchAndParse(f.URL, f.ItemSelector, f.TitleSelector, f.LinkSelector, f.DescriptionSelector, f.PubDateSelector, regexes)
	if err != nil {
		http.Error(w, "Failed to fetch URL: "+err.Error(), http.StatusBadRequest)
		return
	}

	rssXML, err := GenerateRSS(f.Name, f.URL, items)
	if err != nil {
		http.Error(w, "Failed to generate RSS: "+err.Error(), http.StatusInternalServerError)
		return
	}
	f.CachedRSS = rssXML

	id, err := CreateFeed(f)
	if err != nil {
		http.Error(w, "Failed to save feed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	UpdateFeedCache(id, rssXML)

	http.Redirect(w, r, fmt.Sprintf("/feeds/%d", id), http.StatusSeeOther)
}

func handleCreateRSSFeed(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	rssURL := r.FormValue("rss_url")

	if name == "" || rssURL == "" {
		http.Error(w, "Name and RSS URL are required", http.StatusBadRequest)
		return
	}

	f := &Feed{
		Name:     name,
		URL:      rssURL,
		FeedType: "rss",
		RSSURL:   rssURL,
	}

	id, err := CreateFeed(f)
	if err != nil {
		http.Error(w, "Failed to save feed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/feeds/%d", id), http.StatusSeeOther)
}

func handleListFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := ListFeeds()
	if err != nil {
		http.Error(w, "Failed to list feeds: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct{ Feeds []Feed }{Feeds: feeds}
	renderPage(w, "feeds", data)
}

func handleFeedDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid feed ID", http.StatusBadRequest)
		return
	}

	feed, err := GetFeed(id)
	if err != nil {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	data := struct{ Feed *Feed }{Feed: feed}
	renderPage(w, "feed_detail", data)
}

func handleEditFeed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid feed ID", http.StatusBadRequest)
		return
	}

	feed, err := GetFeed(id)
	if err != nil {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	data := struct{ Feed *Feed }{Feed: feed}
	if feed.FeedType == "rss" {
		renderPage(w, "feed_edit_rss", data)
	} else {
		renderPage(w, "feed_edit", data)
	}
}

func handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid feed ID", http.StatusBadRequest)
		return
	}

	// Check existing feed to determine type
	existing, err := GetFeed(id)
	if err != nil {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	if existing.FeedType == "rss" {
		name := r.FormValue("name")
		rssURL := r.FormValue("rss_url")
		if name == "" || rssURL == "" {
			http.Error(w, "Name and RSS URL are required", http.StatusBadRequest)
			return
		}
		existing.Name = name
		existing.RSSURL = rssURL
		existing.URL = rssURL
		err = UpdateFeed(existing)
		if err != nil {
			http.Error(w, "Failed to update feed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/feeds/%d", id), http.StatusSeeOther)
		return
	}

	refreshMinutes, _ := strconv.Atoi(r.FormValue("refresh_minutes"))
	if refreshMinutes < 1 {
		refreshMinutes = 60
	}

	f := &Feed{
		ID:                  id,
		Name:                r.FormValue("name"),
		URL:                 r.FormValue("url"),
		FeedType:            "scrape",
		ItemSelector:        r.FormValue("item_selector"),
		TitleSelector:       r.FormValue("title_selector"),
		LinkSelector:        r.FormValue("link_selector"),
		DescriptionSelector: r.FormValue("description_selector"),
		PubDateSelector:     r.FormValue("pub_date_selector"),
		TitleRegex:          r.FormValue("title_regex"),
		LinkRegex:           r.FormValue("link_regex"),
		DescriptionRegex:    r.FormValue("description_regex"),
		PubDateRegex:        r.FormValue("pub_date_regex"),
		RefreshMinutes:      refreshMinutes,
	}

	if f.Name == "" || f.URL == "" || f.ItemSelector == "" {
		http.Error(w, "Name, URL, and item selector are required", http.StatusBadRequest)
		return
	}

	err = UpdateFeed(f)
	if err != nil {
		http.Error(w, "Failed to update feed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Regenerate RSS with new settings
	regexes := SelectorRegexes{
		Title:       f.TitleRegex,
		Link:        f.LinkRegex,
		Description: f.DescriptionRegex,
		PubDate:     f.PubDateRegex,
	}
	items, err := FetchAndParse(f.URL, f.ItemSelector, f.TitleSelector, f.LinkSelector, f.DescriptionSelector, f.PubDateSelector, regexes)
	if err == nil {
		rssXML, err := GenerateRSS(f.Name, f.URL, items)
		if err == nil {
			UpdateFeedCache(id, rssXML)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/feeds/%d", id), http.StatusSeeOther)
}

func handleFeedRSS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid feed ID", http.StatusBadRequest)
		return
	}

	feed, err := GetFeed(id)
	if err != nil {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(feed.CachedRSS))
}

type opmlXML struct {
	XMLName xml.Name    `xml:"opml"`
	Version string      `xml:"version,attr"`
	Head    opmlHead    `xml:"head"`
	Body    opmlBody    `xml:"body"`
}

type opmlHead struct {
	Title       string `xml:"title"`
	DateCreated string `xml:"dateCreated"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text   string `xml:"text,attr"`
	Title  string `xml:"title,attr"`
	Type   string `xml:"type,attr"`
	XMLURL string `xml:"xmlUrl,attr"`
	HTMLURL string `xml:"htmlUrl,attr"`
}

func handleOPML(w http.ResponseWriter, r *http.Request) {
	feeds, err := ListFeeds()
	if err != nil {
		http.Error(w, "Failed to list feeds: "+err.Error(), http.StatusInternalServerError)
		return
	}

	outlines := make([]opmlOutline, len(feeds))
	for i, f := range feeds {
		xmlURL := fmt.Sprintf("http://%s/feeds/%d/rss?key=%s", r.Host, f.ID, apiKey)
		if f.FeedType == "rss" {
			xmlURL = f.RSSURL
		}
		outlines[i] = opmlOutline{
			Text:    f.Name,
			Title:   f.Name,
			Type:    "rss",
			XMLURL:  xmlURL,
			HTMLURL: f.URL,
		}
	}

	doc := opmlXML{
		Version: "2.0",
		Head: opmlHead{
			Title:       "RSS Feed Generator",
			DateCreated: time.Now().UTC().Format(time.RFC1123Z),
		},
		Body: opmlBody{Outlines: outlines},
	}

	w.Header().Set("Content-Type", "text/x-opml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="feeds.opml"`)
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(doc)
}

func handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid feed ID", http.StatusBadRequest)
		return
	}

	err = DeleteFeed(id)
	if err != nil {
		http.Error(w, "Failed to delete feed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
