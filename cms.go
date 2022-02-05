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

func readMarkdown(path string) (map[string]string, []byte, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	parts := bytes.SplitN(buf, []byte("---\n"), 3)
	headers, err := parseHeaders(string(parts[1]))
	if err != nil {
		return nil, nil, err
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

	return headers, html, nil
}

func readPost(path string) (*Post, error) {
	headers, html, err := readMarkdown(path)
	if err != nil {
		return nil, err
	}
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

type Blog struct {
	settings map[string]string
	pageTmpl *template.Template
	posts    []*Post
}

func load() (*Blog, error) {
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

	return &Blog{
		settings: settings,
		pageTmpl: page,
		posts:    posts,
	}, nil
}

func (blog *Blog) renderFront() error {
	frontTmpl := template.Must(template.Must(blog.pageTmpl.Clone()).ParseFiles("src/templates/frontpage.gotmpl"))
	frontPosts := []interface{}{}
	posts := blog.posts
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
		"title":     blog.settings["title"],
		"extrahead": template.HTML(blog.settings["index_extra_head"]),
		"posts":     frontPosts,
	}
	return renderIfChanged("index.html", frontTmpl, params)
}

func (blog *Blog) renderPosts() error {
	postTmpl := template.Must(template.Must(blog.pageTmpl.Clone()).ParseFiles("src/templates/post.gotmpl"))
	for _, post := range blog.posts {
		params := map[string]interface{}{
			"root":      "../../",
			"title":     blog.settings["title"],
			"pagetitle": blog.settings["title"] + ": " + post.subject,
			"extrahead": template.HTML(blog.settings["index_extra_head"]),
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

func (blog *Blog) renderArchive() error {
	archiveTmpl := template.Must(template.Must(blog.pageTmpl.Clone()).ParseFiles("src/templates/archive.gotmpl"))

	years := []interface{}{}
	var year map[string]interface{}
	var posts []map[string]interface{}
	for _, post := range blog.posts {
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
		"title":     blog.settings["title"],
		"pagetitle": blog.settings["title"] + ": archive",
		"extrahead": template.HTML(blog.settings["index_extra_head"]),
		"years":     years,
	}
	return renderIfChanged("archive.html", archiveTmpl, params)
}

func (blog *Blog) renderFeed() error {
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
		Title: blog.settings["title"],
		ID:    blog.settings["id_base"],
		Link: []Link{
			{Href: blog.settings["link"]},
			{Href: blog.settings["link"] + "atom.xml", Rel: "self"},
		},
		Updated: blog.posts[0].timestamp.Format(time.RFC3339),
		Author: Author{
			Name:  blog.settings["author"],
			Email: blog.settings["email"],
		},
	}

	posts := blog.posts
	if len(posts) > 3 {
		posts = posts[:3]
	}
	for _, post := range posts {
		feed.Entries = append(feed.Entries, Entry{
			ID: blog.settings["id_base"] + "/" +
				post.timestamp.Format("2006-01-02") + "/" +
				post.filename,
			Updated: post.timestamp.Format(time.RFC3339),
			Title:   post.subject,
			Link:    Link{Href: blog.settings["link"] + post.htmlPath()},
			Content: Content{Type: "html", Body: string(post.html)},
		})
	}

	buf, err := xml.Marshal(feed)
	if err != nil {
		return err
	}

	return writeIfChanged("atom.xml", buf)
}

func renderBlog() error {
	blog, err := load()
	if err != nil {
		return err
	}
	if err := blog.renderPosts(); err != nil {
		return err
	}
	if err := blog.renderFront(); err != nil {
		return err
	}
	if err := blog.renderArchive(); err != nil {
		return err
	}
	if err := blog.renderFeed(); err != nil {
		return err
	}

	return nil
}

func renderPage(tmpl *template.Template, path string) error {
	headers, html, err := readMarkdown(path)
	if err != nil {
		return err
	}

	htmlPath := strings.TrimSuffix(path, ".md") + ".html"

	slashes := strings.Count(path, "/")
	root := strings.Repeat("../", slashes)
	params := map[string]interface{}{
		"title":      headers["title"],
		"customhead": template.HTML(headers["customhead"]),
		"root":       root,
		"frontpage":  headers["frontpage"],
		"content":    template.HTML(html),
		"lastupdate": headers["lastupdate"],
	}
	return renderIfChanged(htmlPath, tmpl, params)
}

func renderSite() error {
	tmpl := template.Must(template.ParseFiles("site/page.gotmpl"))

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			basename := filepath.Base(path)
			if basename == ".git" || basename == "_darcs" || basename == "blog" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		return renderPage(tmpl, path)
	})
	return err
}

func run(args []string) error {
	usage := fmt.Errorf("usage: cms {blog|site}")
	if len(args) != 1 {
		return usage
	}
	switch args[0] {
	case "blog":
		return renderBlog()
	case "site":
		return renderSite()
	default:
		return usage
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}
