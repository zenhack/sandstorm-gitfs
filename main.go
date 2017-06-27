package main

import (
	"html/template"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"strings"

	"github.com/gorilla/mux"

	"zenhack.net/go/sandstorm-gitfs/git"
)

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("FATAL: environment variable %s must be set and "+
			"non-empty.", key)
	}
	return val
}

var tpls = template.Must(template.ParseGlob("templates/*.html"))

type TemplateArg struct {
	RepoName string
	Files    []git.TreeEntry
}

func ensureRepo(gitProjectRoot string) {
	err := os.MkdirAll(gitProjectRoot+"/r", 0700)
	if err != nil {
		log.Fatal("Creating %s: %v", gitProjectRoot, err)
	}
	repoPath := gitProjectRoot + "/r/_repo.git"
	fi, err := os.Stat(repoPath)
	if !(err == nil && fi.IsDir()) {
		err = exec.Command("git", "init", "--bare", repoPath).Run()
		if err != nil {
			log.Fatal("Creating repository:", err)
		}
		err = exec.Command("git", "--git-dir="+repoPath, "config",
			"http.receivepack", "true").Run()
		if err != nil {
			log.Fatal("Enabling push:", err)
		}
	}
}

func main() {
	ensureRepo(mustEnv("GIT_PROJECT_ROOT"))

	r := mux.NewRouter()
	r.Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tpls.ExecuteTemplate(w, "test.html", &TemplateArg{
			RepoName: "My silly project",
			Files: []git.TreeEntry{
				{Name: "README.md", Type: "blob"},
				{Name: "src", Type: "tree"},
				{Name: ".gitignore", Type: "blob"},
			},
		})
	})
	// Handle requests from git itself. We alias /r/* to the repo, so the
	// user can clone it by any name.
	r.PathPrefix("/r/{reponame}").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gitHandler := &cgi.Handler{
			Path:       "/usr/bin/git",
			Args:       []string{"http-backend"},
			InheritEnv: []string{"GIT_PROJECT_ROOT"},
			Env: []string{
				"GIT_HTTP_EXPORT_ALL=",
			},
		}
		log.Print(req.URL.Path)
		parts := strings.SplitN(req.URL.Path, "/", 4)
		parts[2] = "_repo.git"
		req.URL.Path = strings.Join(parts, "/")
		gitHandler.ServeHTTP(w, req)
	})
	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}
