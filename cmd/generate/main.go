package main

import (
	"bytes"
	"flag"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"gopkg.in/yaml.v2"
)

var (
	configPath        = flag.String("config", "global.config.yaml", "")
	markdownPostsPath = flag.String("markdown", "blog", "")
	output            = flag.String("output", "docs", "")
	templates         = flag.String("templates", "templates/*", "")
	niceFormat        = "2006-01-02T15:04:05-0700"
)

type Config struct {
	SocialLinks []*SocialLink `yaml:"social_links"`
	Author      string        `yaml:"author"`
	Email       string        `yaml:"email"`
	About       string        `yaml:"about"`
}

type SocialLink struct {
	Icon  string `yaml:"icon"`
	Url   string `yaml:"url"`
	Color string `yaml:"color"`
}

type Page struct {
	Tag             string
	About           string
	SocialLinks     []*SocialLink
	Contents        template.HTML
	MetaTitle       string
	MetaAuthor      string
	MetaImage       string
	MetaDate        string
	MetaDescription string
	MetaTwitter     string
}

type Index struct {
	Tag   string
	Posts []*Post
}

type Post struct {
	Title       string
	Date        time.Time
	DateString  string
	Preview     string
	Url         string
	Description string
	Tags        []string
	Contents    template.HTML
}

func main() {
	flag.Parse()
	t, err := template.ParseGlob(*templates)
	if err != nil {
		panic(err)
	}
	configFile, err := os.Open(*configPath)
	if err != nil {
		panic(err)
	}
	defer configFile.Close()
	encConfig, err := io.ReadAll(configFile)
	if err != nil {
		panic(err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(encConfig, cfg); err != nil {
		panic(err)
	}
	page := &Page{
		About:       cfg.About,
		SocialLinks: cfg.SocialLinks,
		MetaAuthor:  cfg.Author,
	}
	log.Println("Rendering index page for posts")
	if err := renderIndexPage(t, page, *markdownPostsPath, *output); err != nil {
		panic(err)
	}
	log.Println("Rendering tag pages for posts")
	if err := renderTagPages(t, page, *markdownPostsPath, *output); err != nil {
		panic(err)
	}
	log.Println("Rendering individual blog posts")
	if err := renderBlogPosts(t, page, *markdownPostsPath, *output); err != nil {
		panic(err)
	}
}

// Renders the index page for the website.
func renderIndexPage(t *template.Template, page *Page, postsPath, outputPath string) error {
	blogPostFiles, err := parseBlogPostFileNames(postsPath)
	if err != nil {
		return err
	}
	posts := make([]*Post, 0)
	for _, bp := range blogPostFiles {
		post, err := generateBlogPostHtmlFromMarkdown(bp)
		if err != nil {
			return err
		}
		posts = append(posts, post)
	}
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})
	indexContents := bytes.NewBuffer(nil)
	if err := t.Lookup("index").Execute(indexContents, &Index{Posts: posts}); err != nil {
		return err
	}
	page.MetaTitle = "Blog"
	page.MetaAuthor = "Raul Jordan"
	page.MetaDate = time.Now().Format(niceFormat)
	page.MetaDescription = "Raul Jordan's technical blog"
	page.Contents = template.HTML(indexContents.String())
	outputPath = filepath.Join(outputPath, "index.html")
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Lookup("base").Execute(f, page)
}

// Renders the tag pages for the website.
func renderTagPages(t *template.Template, page *Page, postsPath, outputPath string) error {
	blogPostFiles, err := parseBlogPostFileNames(postsPath)
	if err != nil {
		return err
	}
	posts := make([]*Post, 0)
	for _, bp := range blogPostFiles {
		post, err := generateBlogPostHtmlFromMarkdown(bp)
		if err != nil {
			return err
		}
		posts = append(posts, post)
	}
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	foundTags := make(map[string][]*Post)
	for _, p := range posts {
		for _, t := range p.Tags {
			foundTags[t] = append(foundTags[t], p)
		}
	}

	for tag, taggedPosts := range foundTags {
		indexContents := bytes.NewBuffer(nil)
		if err := t.Lookup("tags").Execute(indexContents, &Index{Posts: taggedPosts, Tag: tag}); err != nil {
			return err
		}
		page.MetaTitle = tag
		page.MetaAuthor = "Raul Jordan"
		page.MetaDate = time.Now().Format(niceFormat)
		page.MetaDescription = "Raul Jordan's technical blog"
		page.Contents = template.HTML(indexContents.String())
		out := filepath.Join(outputPath, "tag", tag, "index.html")
		dirPath := filepath.Dir(out)
		fullPath, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(fullPath, dirPath), os.ModePerm); err != nil {
			return err
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := t.Lookup("base").Execute(f, page); err != nil {
			return err
		}
	}
	return nil
}

// Renders all blog posts into static files.
func renderBlogPosts(t *template.Template, page *Page, postsPath, outputPath string) error {
	blogPostFiles, err := parseBlogPostFileNames(postsPath)
	if err != nil {
		return err
	}
	for _, blogPost := range blogPostFiles {
		if err := renderIndividualBlogPost(t, page, blogPost, outputPath); err != nil {
			return err
		}
	}
	return nil
}

// Renders individual blog posts by rendering HTML from markdown and executing
// from a base post template.
func renderIndividualBlogPost(t *template.Template, page *Page, blogPostFile, outputPath string) error {
	blogPostDir := filepath.Dir(blogPostFile)
	blogPostName := strings.TrimPrefix(blogPostFile, blogPostDir)
	blogPostName = strings.Replace(blogPostName, "-", "/", 3)
	title := strings.TrimSuffix(blogPostName, ".md")
	dirPath := filepath.Dir(title)
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(wd, outputPath, dirPath), os.ModePerm); err != nil {
		return err
	}
	outputPath = filepath.Join(outputPath, title+".html")

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	postTpl := t.Lookup("post")
	post, err := generateBlogPostHtmlFromMarkdown(blogPostFile)
	if err != nil {
		return err
	}
	renderedPost := bytes.NewBuffer(nil)
	if err := postTpl.Execute(renderedPost, post); err != nil {
		return err
	}
	page.Contents = template.HTML(renderedPost.String())
	page.MetaDate = post.Date.Format(niceFormat)
	page.MetaTitle = post.Title
	page.MetaDescription = post.Description
	return t.Lookup("base").Execute(f, page)
}

// Parse the names of blog post files with .md extensions from a directory path.
func parseBlogPostFileNames(postsPath string) ([]string, error) {
	blogPostFiles := make([]string, 0)
	if err := filepath.Walk(postsPath, func(path string, info fs.FileInfo, err error) error {
		if path == postsPath {
			return nil
		}
		if !strings.Contains(path, ".md") {
			return nil
		}
		blogPostFiles = append(blogPostFiles, path)
		return nil
	}); err != nil {
		return blogPostFiles, err
	}
	return blogPostFiles, nil
}

// Generates the HTML for a blog post from a markdown file.
func generateBlogPostHtmlFromMarkdown(blogPostPath string) (*Post, error) {
	f, err := os.Open(blogPostPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	enc, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
			),
		),
	)

	buf := bytes.NewBuffer(nil)
	ctx := parser.NewContext()
	if err := markdown.Convert(enc, buf, parser.WithContext(ctx)); err != nil {
		return nil, err
	}
	metaData := meta.Get(ctx)

	var title string
	var preview string
	var desc string
	var tags []string
	var date time.Time
	var dateString string
	if tl, ok := metaData["title"].(string); ok {
		title = tl
	}
	if pr, ok := metaData["preview"].(string); ok {
		preview = pr
	}
	if d, ok := metaData["description"].(string); ok {
		desc = d
	}
	if d, ok := metaData["date"].(string); ok {
		dateString = d
		td, err := time.Parse("2006-Jan-02", d)
		if err != nil {
			return nil, err
		}
		date = td
	}
	if tg, ok := metaData["tags"].([]interface{}); ok {
		tagItems := make([]string, len(tg))
		for i, ttg := range tg {
			if item, ok := ttg.(string); ok {
				tagItems[i] = item
			}
		}
		tags = tagItems
	}
	blogPostDir := filepath.Dir(blogPostPath)
	blogPostName := strings.TrimPrefix(blogPostPath, blogPostDir)
	url := strings.TrimSuffix(blogPostName, ".md")
	url = strings.Replace(url, "-", "/", 3)
	return &Post{
		Title:       title,
		Preview:     preview,
		Date:        date,
		DateString:  dateString,
		Description: desc,
		Tags:        tags,
		Url:         url + ".html",
		Contents:    template.HTML(buf.String()),
	}, nil
}
