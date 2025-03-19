package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template/parse"
)

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := createFilePath(p.Title)
	return os.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := createFilePath(title)
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func createFilePath(title string) string {
	return "data/" + title + ".txt"
}

type Layout struct {
	Title     string
	ChildArgs any
}

var layoutTemplate = template.Must(template.ParseFiles("tmpl/layout.html"))

func renderTemplateByLayout(w http.ResponseWriter, l Layout, tmplTree *parse.Tree) {
	t, err := template.Must(layoutTemplate.Clone()).AddParseTree("main", tmplTree)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.ExecuteTemplate(w, "layout.html", l)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

var templates = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/view.html", "tmpl/list.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	l := Layout{
		Title:     fmt.Sprintf("%s: %s", tmpl, p.Title),
		ChildArgs: p,
	}

	t := templates.Lookup(tmpl + ".html").Tree
	renderTemplateByLayout(w, l, t)
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	dirs, err := os.ReadDir("data")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pages := make([]Page, 0, len(dirs))

	for _, d := range dirs {
		if d.IsDir() {
			continue
		}

		info, err := d.Info()
		if err != nil {
			fmt.Println(err)
			continue
		}

		fName := info.Name()

		if fName == ".gitkeep" {
			continue
		}

		title := strings.TrimSuffix(fName, filepath.Ext(fName))
		p, err := loadPage(title)
		if err != nil {
			fmt.Println(err)
			continue
		}
		pages = append(pages, *p)
	}

	l := Layout{
		Title: "List",
		ChildArgs: struct {
			Pages []Page
		}{Pages: pages},
	}

	t := templates.Lookup("list.html").Tree
	renderTemplateByLayout(w, l, t)
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/home", http.StatusMovedPermanently)
}

func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/list", listHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
