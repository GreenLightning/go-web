package web

import (
	"html/template"
	"io"
	"log"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type FuncMap template.FuncMap

type Renderer struct {
	directory string
	base      *template.Template
	templates *template.Template
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}) error {
	return r.templates.ExecuteTemplate(w, name, data)
}

// NewRenderer parses the templates from directory.
// Subdirectories are not supported at the moment,
// because the template package identifies templates
// by filename alone.
func NewRenderer(directory string) *Renderer {
	return NewRendererWithFunctions(directory, nil)
}

func NewRendererWithFunctions(directory string, functions FuncMap) *Renderer {
	r := new(Renderer)
	r.directory = directory
	pattern := strings.TrimRight(directory, "/") + "/*"
	r.base = template.Must(template.New("").Funcs(template.FuncMap(functions)).ParseGlob(pattern))
	r.templates = template.Must(r.base.Clone())
	return r
}

func (r *Renderer) WatchTemplateDirectory() {
	watcher, err := fsnotify.NewWatcher() // @Leak: Close watcher.
	if err != nil {
		log.Println("[web][render][warning] failed to create template watcher:", err)
		return
	}

	err = watcher.Add(r.directory)
	if err != nil {
		log.Println("[web][render][warning] failed to watch template directory:", err)
		return
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write != 0 {
					updated, err := r.base.ParseFiles(event.Name)
					if err != nil {
						log.Printf("[web][render][warning] failed to reload template file (%s): %v", event.Name, err)
					} else {
						r.base = updated
						r.templates = template.Must(updated.Clone())
						log.Println("[web][render][info] updated template file:", event.Name)
					}
				}
			case err := <-watcher.Errors:
				log.Println("[web][render][warning] template watcher error:", err)
			}
		}
	}()
}
