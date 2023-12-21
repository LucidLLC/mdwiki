package main

import (
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v3"

	cp "github.com/otiai10/copy"

	templateHtml "html/template"
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
	DefaultPageTemplate = templateHtml.Must(templateHtml.ParseFiles("template/page.html"))
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

// RenderInput is the actual struct that gets passed in when rendering the template
type RenderInput struct {
	Entries []Entry

	Title   string
	Content templateHtml.HTML
}

// CompiledPage is a page that has rendered HTML and the title from the config.
type CompiledPage struct {
	Original *Page
	Title    string
	Content  string
}

// this uses Go's HTML template engine to
func (p *CompiledPage) RenderTo(entries []Entry, w io.Writer) error {
	return DefaultPageTemplate.Execute(w, &RenderInput{
		Entries: entries,
		Title:   p.Title,
		Content: templateHtml.HTML(p.Content),
	})
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

func (p *Page) CompileDirectory() string {
	if p.Type == Index {
		return CompiledDirectory
	}
	dir, _ := filepath.Split(p.ContentPath)
	dir = filepath.Base(dir)
	return filepath.Join(CompiledDirectory, dir)
}

func (p *Page) CompilePath() string {
	if p.Type == Index {
		return filepath.Join(CompiledDirectory, CompiledContentFile)
	}

	dir, _ := filepath.Split(p.ContentPath)
	dir = filepath.Base(dir)
	return filepath.Join(CompiledDirectory, dir, CompiledContentFile)
}

func (p *Page) HttpPath() string {
	if p.Type == Index {
		return "/"
	}

	dir, _ := filepath.Split(p.ContentPath)
	dir = filepath.Base(dir)
	return "/" + dir
}

func (p *Page) String() string {
	return p.ContentPath
}

func (p *Page) Compile() (*CompiledPage, error) {
	config, err := os.ReadFile(p.ConfigPath)

	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(p.ContentPath)

	if err != nil {
		return nil, err
	}

	log.Println(string(content), string(config))

	var pageConfig PageConfig

	if err := yaml.Unmarshal(config, &pageConfig); err != nil {
		return nil, err
	}

	renderedMarkdown := markdown.ToHTML(content, parser.New(), html.NewRenderer(html.RendererOptions{
		Flags: html.CommonFlags | html.LazyLoadImages,
	}))

	return &CompiledPage{
		Original: p,
		Title:    pageConfig.Title,
		Content:  string(renderedMarkdown),
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
	pages := CollectPages(ParentDirectory)
	compiledPages := make([]*CompiledPage, len(pages))

	// we need to do a few rounds - first 'compile' the pages.
	// then collect the entries.
	for i, p := range pages {
		compiled, err := p.Compile()

		if err != nil {
			log.Fatalln(err)
		}

		compiledPages[i] = compiled
	}

	// collect the entries for each page

	entriesForPage := make(map[*CompiledPage][]Entry)

	log.Println(len(compiledPages), len(pages), pages)
	for i, p := range compiledPages {
		entriesForPage[p] = make([]Entry, len(compiledPages))

		for j, op := range compiledPages {
			entriesForPage[p][j] = Entry{
				Title:  op.Title,
				Link:   op.Original.HttpPath(),
				Active: i == j,
			}
		}

		// Create the compile directories
		os.MkdirAll(p.Original.CompileDirectory(), os.ModePerm)

		// create the compiled file
		f, err := os.Create(p.Original.CompilePath())

		if err != nil {
			log.Fatalln(err)
		}

		// attempt to render to the given file
		if err := p.RenderTo(entriesForPage[p], f); err != nil {
			log.Fatalln(err)
		}

		f.Close() // close the file
	}

	// Copy the assets from the parent path to compiled path
	cp.Copy("./assets/", filepath.Join(CompiledDirectory, "assets"))
}
