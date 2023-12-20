package main

import (
	"io/fs"
	"log"
	"path/filepath"
	"strings"
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

type PageType int

const (
	Index PageType = iota
	SubPage
)

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
	pages := CollectPages("pages")

	log.Println(pages[0].CompiledPath(), pages[1].CompiledPath())
}
