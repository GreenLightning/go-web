package web

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	htmltemplate "html/template"
	texttemplate "text/template"

	"github.com/fsnotify/fsnotify"
)

type FuncMap map[string]interface{}

type Renderer struct {
	directory string

	// @Note: The template gets locked during the first call to execute.
	// When watching the template directory, we want to parse and execute
	// in parallel, so we have to keep a clean base copy of the template
	// for parsing and the regular template which is used for executing.
	textbase      *texttemplate.Template
	texttemplates *texttemplate.Template

	htmlbase      *htmltemplate.Template
	htmltemplates *htmltemplate.Template
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}) error {
	if isText(name) {
		return r.texttemplates.ExecuteTemplate(w, name, data)
	} else {
		return r.htmltemplates.ExecuteTemplate(w, name, data)
	}
}

// NewRenderer parses the templates from the given directory.
//
// Subdirectories are not supported at the moment, because the template
// package identifies templates by filename alone.
//
// The text/template package is used for files ending in .text.ext.
// All other files are handled by the html/template package.
func NewRenderer(directory string) *Renderer {
	return NewRendererWithFunctions(directory, nil)
}

func NewRendererWithFunctions(directory string, functions FuncMap) *Renderer {
	textfiles, htmlfiles, err := readFiles(directory)
	if err != nil {
		log.Println("[renderer] error: failed to read template directory:", err)
	}

	r := new(Renderer)
	r.directory = directory

	r.textbase = texttemplate.New("").Funcs(texttemplate.FuncMap(functions))
	if len(textfiles) != 0 {
		r.textbase = texttemplate.Must(r.textbase.ParseFiles(textfiles...))
	}
	r.texttemplates = texttemplate.Must(r.textbase.Clone())

	r.htmlbase = htmltemplate.New("").Funcs(htmltemplate.FuncMap(functions))
	if len(htmlfiles) != 0 {
		r.htmlbase = htmltemplate.Must(r.htmlbase.ParseFiles(htmlfiles...))
	}
	r.htmltemplates = htmltemplate.Must(r.htmlbase.Clone())
	return r
}

func readFiles(directory string) (textfiles []string, htmlfiles []string, err error) {
	infos, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, nil, err
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		filename := filepath.Join(directory, info.Name())
		if isText(filename) {
			textfiles = append(textfiles, filename)
		} else {
			htmlfiles = append(htmlfiles, filename)
		}
	}

	return
}

func (r *Renderer) WatchTemplateDirectory() {
	watcher, err := fsnotify.NewWatcher() // @Leak: Close watcher.
	if err != nil {
		log.Println("[renderer] warning: failed to create template watcher:", err)
		return
	}

	err = watcher.Add(r.directory)
	if err != nil {
		log.Println("[renderer] warning: failed to watch template directory:", err)
		return
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write != 0 {
					var err error

					if isText(event.Name) {
						var updated *texttemplate.Template
						updated, err = r.textbase.ParseFiles(event.Name)
						if err == nil {
							r.textbase = updated
							r.texttemplates = texttemplate.Must(updated.Clone())
						}
					} else {
						var updated *htmltemplate.Template
						updated, err = r.htmlbase.ParseFiles(event.Name)
						if err == nil {
							r.htmlbase = updated
							r.htmltemplates = htmltemplate.Must(updated.Clone())
						}
					}

					if err != nil {
						log.Printf("[renderer] warning: failed to reload template file: %s: %v", event.Name, err)
					} else {
						log.Println("[renderer] info: updated template file:", event.Name)
					}
				}

			case err := <-watcher.Errors:
				log.Println("[renderer] warning: template watcher error:", err)
			}
		}
	}()
}

func isText(filename string) bool {
	return ext2(filename) == ".text"
}

// ext2 returns the paths second extension.
//
// The second extension is the one between the second to last and the last dot
// in the final elementq of the path. It contains a dot on the left, but not
// on the right. It is empty if the final element does not contain two dots.
//
// Examples:
// "filename.a.b.c" => ".b"
// "filename.ext" => ""
func ext2(path string) string {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			for j := i - 1; j >= 0 && !os.IsPathSeparator(path[j]); j-- {
				if path[j] == '.' {
					return path[j:i]
				}
			}
		}
	}

	return ""
}
