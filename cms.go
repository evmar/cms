package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

type Post struct {
	subject   string
	summary   string
	filename  string
	timestamp time.Time
	html      []byte
}

func (p *Post) htmlPath() string {
	return p.timestamp.Format("2006/01/") + p.filename + ".html"
}

func parseHeaders(text string) (map[string]string, error) {
	headers := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad header line %q", line)
		}
		headers[parts[0]] = parts[1]
	}
	return headers, nil
}

func readSettings() (map[string]string, error) {
	buf, err := os.ReadFile("src/settings")
	if err != nil {
		return nil, err
	}
	return parseHeaders(string(buf))
}

func readPost(path string) (*Post, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := bytes.SplitN(buf, []byte("---\n"), 3)
	headers, err := parseHeaders(string(parts[1]))
	if err != nil {
		return nil, err
	}
	html := markdown.ToHTML(parts[2], nil, html.NewRenderer(html.RendererOptions{
		Flags: html.FlagsNone,
	}))
	// Hacks to make the output match the previous rendering:
	html = bytes.ReplaceAll(html, []byte(" --- "), []byte(" &mdash; "))
	html = bytes.ReplaceAll(html, []byte(" -- "), []byte(" &mdash; "))
	html = bytes.ReplaceAll(html, []byte(" --\n"), []byte(" &mdash;\n"))
	html = bytes.ReplaceAll(html, []byte("\n--"), []byte("\n&mdash;"))
	html = bytes.ReplaceAll(html, []byte("&quot;"), []byte("\""))
	html = bytes.ReplaceAll(html, []byte(">\n\n"), []byte(">\n"))
	html = bytes.ReplaceAll(html, []byte("<hr>"), []byte("<hr />"))
	html = bytes.ReplaceAll(html, []byte("<li><p>"), []byte("<li>\n<p>"))
	html = bytes.ReplaceAll(html, []byte("</p></li>"), []byte("</p>\n</li>"))
	html = bytes.Trim(html, " \n")

	var timestamp time.Time
	if ts := headers["Timestamp"]; ts != "" {
		timestamp, err = time.Parse("2006/01/02 15:04", headers["Timestamp"])
	} else {
		timestamp, err = time.Parse("2006/01/02", headers["Date"])
	}
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(path)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	post := &Post{
		subject:   headers["Subject"],
		summary:   headers["Summary"],
		filename:  filename,
		timestamp: timestamp,
		html:      html,
	}

	return post, nil
}

func readPosts() ([]*Post, error) {
	posts := []*Post{}
	err := filepath.WalkDir("src/posts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		post, err := readPost(path)
		if err != nil {
			return err
		}
		posts = append(posts, post)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func writeIfChanged(path string, data []byte) error {
	old, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if bytes.Equal(old, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0666); err != nil {
		return err
	}
	fmt.Printf(" * %s\n", path)
	return nil
}

func renderIfChanged(path string, tmpl *template.Template, params interface{}) error {
	buf := bytes.Buffer{}
	err := tmpl.Execute(&buf, params)
	if err != nil {
		return err
	}
	return writeIfChanged(path, buf.Bytes())
}

type Site struct {
	settings map[string]string
	pageTmpl *template.Template
	posts    []*Post
}

func load() (*Site, error) {
	settings, err := readSettings()
	if err != nil {
		return nil, err
	}

	page := template.Must(template.ParseFiles("src/templates/page.gotmpl"))

	posts, err := readPosts()
	if err != nil {
		return nil, err
	}
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].timestamp.After(posts[j].timestamp)
	})

	return &Site{
		settings: settings,
		pageTmpl: page,
		posts:    posts,
	}, nil
}

func (site *Site) renderFront() error {
	frontTmpl := template.Must(template.Must(site.pageTmpl.Clone()).ParseFiles("src/templates/frontpage.gotmpl"))
	frontPosts := []interface{}{}
	posts := site.posts
	if len(posts) > 10 {
		posts = posts[:10]
	}
	for _, post := range posts {
		frontPosts = append(frontPosts, map[string]interface{}{
			"title":   post.subject,
			"summary": post.summary,
			"date":    post.timestamp.Format("2006/01/02"),
			"path":    post.htmlPath(),
		})
	}
	params := map[string]interface{}{
		"title":     site.settings["title"],
		"extrahead": template.HTML(site.settings["index_extra_head"]),
		"posts":     frontPosts,
	}
	return renderIfChanged("index.html", frontTmpl, params)
}

func (site *Site) renderPosts() error {
	postTmpl := template.Must(template.Must(site.pageTmpl.Clone()).ParseFiles("src/templates/post.gotmpl"))
	for _, post := range site.posts {
		params := map[string]interface{}{
			"root":      "../../",
			"title":     site.settings["title"],
			"pagetitle": site.settings["title"] + ": " + post.subject,
			"extrahead": template.HTML(site.settings["index_extra_head"]),
			"post": map[string]interface{}{
				"url":     "",
				"title":   post.subject,
				"date":    post.timestamp.Format("January 02, 2006"),
				"content": template.HTML(post.html),
			},
		}
		if err := renderIfChanged(post.htmlPath(), postTmpl, params); err != nil {
			return err
		}
	}
	return nil
}

func (site *Site) renderArchive() error {
	archiveTmpl := template.Must(template.Must(site.pageTmpl.Clone()).ParseFiles("src/templates/archive.gotmpl"))

	years := []interface{}{}
	var year map[string]interface{}
	var posts []map[string]interface{}
	for _, post := range site.posts {
		if year == nil || post.timestamp.Year() != year["year"].(int) {
			if year != nil {
				year["posts"] = posts
				posts = nil
			}
			year = map[string]interface{}{
				"year": post.timestamp.Year(),
			}
			years = append(years, year)
		}
		posts = append(posts, map[string]interface{}{
			"path":  post.htmlPath(),
			"title": post.subject,
			"date":  post.timestamp.Format("January 02"),
		})
	}
	year["posts"] = posts

	params := map[string]interface{}{
		"root":      "./",
		"title":     site.settings["title"],
		"pagetitle": site.settings["title"] + ": archive",
		"extrahead": template.HTML(site.settings["index_extra_head"]),
		"years":     years,
	}
	return renderIfChanged("archive.html", archiveTmpl, params)
}

func (site *Site) renderFeed() error {
	type Link struct {
		XMLName xml.Name `xml:"link"`
		Rel     string   `xml:"rel,attr,omitempty"`
		Href    string   `xml:"href,attr"`
	}
	type Author struct {
		XMLName xml.Name `xml:"author"`
		Name    string   `xml:"name"`
		Email   string   `xml:"email"`
	}
	type Content struct {
		XMLName xml.Name `xml:"content"`
		Type    string   `xml:"type,attr"`
		Body    string   `xml:",chardata"`
	}
	type Entry struct {
		XMLName xml.Name `xml:"entry"`
		ID      string   `xml:"id"`
		Updated string   `xml:"updated"`
		Title   string   `xml:"title"`
		Link    Link
		Content Content
	}
	type Feed struct {
		XMLName xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
		Title   string   `xml:"title"`
		ID      string   `xml:"id"`
		Link    []Link
		Updated string `xml:"updated"`
		Author  Author
		Entries []Entry
	}

	feed := Feed{
		Title: site.settings["title"],
		ID:    site.settings["id_base"],
		Link: []Link{
			{Href: site.settings["link"]},
			{Href: site.settings["link"] + "atom.xml", Rel: "self"},
		},
		Updated: site.posts[0].timestamp.Format(time.RFC3339),
		Author: Author{
			Name:  site.settings["author"],
			Email: site.settings["email"],
		},
	}

	posts := site.posts
	if len(posts) > 3 {
		posts = posts[:3]
	}
	for _, post := range posts {
		feed.Entries = append(feed.Entries, Entry{
			ID: site.settings["id_base"] + "/" +
				post.timestamp.Format("2006-01-02") + "/" +
				post.filename,
			Updated: post.timestamp.Format(time.RFC3339),
			Title:   post.subject,
			Link:    Link{Href: site.settings["link"] + post.htmlPath()},
			Content: Content{Type: "html", Body: string(post.html)},
		})
	}

	buf, err := xml.Marshal(feed)
	if err != nil {
		return err
	}

	return writeIfChanged("atom.xml", buf)
}

func run() error {
	site, err := load()
	if err != nil {
		return err
	}
	if err := site.renderPosts(); err != nil {
		return err
	}
	if err := site.renderFront(); err != nil {
		return err
	}
	if err := site.renderArchive(); err != nil {
		return err
	}
	if err := site.renderFeed(); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}
