package main

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v3"
)

const (
	// This is the index page
	IndexFile   = "index.md"
	ContentFile = "content.md"
	ConfigFile  = "config.yml"

	ParentDirectory = "pages"

	CompiledDirectory   = "compiled"
	CompiledContentFile = "index.html"
)

var (
	HtmlMarkdownRenderer = html.NewRenderer(html.RendererOptions{
		Flags: html.CommonFlags | html.LazyLoadImages,
	})

	DefaultMarkdownParser = parser.New()
)

type PageType int

const (
	Index PageType = iota
	SubPage
)

type PageConfig struct {
	Title string `yaml:"title"`
}

type Entry struct {
	Title  string
	Link   string
	Active bool
}

type CompiledTemplate struct {
	Title   string
	Content string
}

type RenderInput struct {
	Entries []Entry

	Title   string
	Content string
}

func (*CompiledTemplate) RenderTo(entries []Entry, w io.Writer) error {
	return nil
}

/*
	  Directory structure:

	  assets/
	    css/     # optional
		images/
	  pages/
	    path/
		  config.yml
		  content.md
	    index.md
	  template/
	    pages/
		  path.html # if path was banking, template would be banking.html
	    page.html # default page template.
*/
type Page struct {
	Type        PageType
	ConfigPath  string
	ContentPath string
}

func (p *Page) CompiledPath() string {
	if p.Type == Index {
		return filepath.Join(CompiledDirectory, CompiledContentFile)
	}

	dir, _ := filepath.Split(p.ContentPath)
	dir = filepath.Base(dir)
	return filepath.Join(CompiledDirectory, dir, CompiledContentFile)
}

func (p *Page) String() string {
	return p.ContentPath
}

func (p *Page) Compile() (*CompiledTemplate, error) {
	config, err := os.ReadFile(p.ConfigPath)

	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(p.ContentPath)

	if err != nil {
		return nil, err
	}

	var pageConfig PageConfig
	if err := yaml.Unmarshal(config, pageConfig); err != nil {
		return nil, err
	}

	renderedMarkdown := markdown.ToHTML(content, DefaultMarkdownParser, HtmlMarkdownRenderer)

	return &CompiledTemplate{
		Title:   pageConfig.Title,
		Content: string(renderedMarkdown),
	}, nil
}

// CollectPages walks through a directory and collects all the valid pages
func CollectPages(directory string) []*Page {

	var pages []*Page

	filepath.WalkDir(directory, func(path string, file fs.DirEntry, err error) error {
		if !file.IsDir() {
			if strings.EqualFold(ContentFile, file.Name()) || strings.EqualFold(IndexFile, file.Name()) {
				parentDir := filepath.Dir(path)
				page := &Page{
					Type:        SubPage,
					ConfigPath:  filepath.Join(parentDir, ConfigFile),
					ContentPath: path,
				}

				if strings.EqualFold(file.Name(), IndexFile) && strings.EqualFold(directory, parentDir) {
					page.Type = Index
				}

				pages = append(pages, page)
			}
		}
		return nil
	})

	return pages
}

func main() {
}
