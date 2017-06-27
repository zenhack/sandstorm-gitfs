package main

import (
	"html/template"
	"net/http"

	"zenhack.net/go/sandstorm-gitfs/git"
)

var tpls = template.Must(template.ParseGlob("templates/*.html"))

type TemplateArg struct {
	RepoName string
	Files    []git.TreeEntry
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		tpls.ExecuteTemplate(w, "test.html", &TemplateArg{
			RepoName: "My silly project",
			Files: []git.TreeEntry{
				{Name: "README.md", Type: "blob"},
				{Name: "src", Type: "tree"},
				{Name: ".gitignore", Type: "blob"},
			},
		})
	})
	http.ListenAndServe(":8080", nil)
}
