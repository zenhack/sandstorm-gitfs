package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"strings"

	"github.com/gorilla/mux"

	"zenhack.net/go/sandstorm-gitfs/git"

	"zenhack.net/go/sandstorm-filesystem/filesystem"
	"zenhack.net/go/sandstorm-filesystem/filesystem/httpfs"

	grain_capnp "zenhack.net/go/sandstorm/capnp/grain"
	"zenhack.net/go/sandstorm/grain"
	"zenhack.net/go/sandstorm/websession"

	"zombiezen.com/go/capnproto2/pogs"
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

func ensureRepo(gitProjectRoot string) *git.Git {
	err := os.MkdirAll(gitProjectRoot+"/r", 0700)
	if err != nil {
		log.Fatal("Creating %s: %v", gitProjectRoot, err)
	}
	repoPath := gitProjectRoot + "/r/_repo.git"
	fi, err := os.Stat(repoPath)
	if !(err == nil && fi.IsDir()) {
		g, err := git.InitBare(repoPath)
		if err != nil {
			log.Fatal("Creating repository:", err)
		}
		err = g.SetConfig("http.receivepack", "true")
		if err != nil {
			log.Fatal("Enabling push:", err)
		}
		return g
	}
	return &git.Git{GitDir: repoPath}
}

func main() {
	g := ensureRepo(mustEnv("GIT_PROJECT_ROOT"))

	r := mux.NewRouter()
	r.Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Location", "/browse/master")
		w.WriteHeader(http.StatusSeeOther) // TODO: correct status?
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
	r.PathPrefix("/browse/{commit}").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		commit := mux.Vars(req)["commit"]
		h, err := g.GetCommitTree(commit)
		if err != nil {
			// TODO: be more methodical. For now, we just assume any error is
			// a missing commit/branch.
			log.Print(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		tree := filesystem.Directory_ServerToClient(&Dir{
			TreeEntry: git.TreeEntry{
				Hash: h,
				Type: "tree",
			},
			g: *g,
		})
		http.FileServer(&PrefixStripper{
			FS:     &httpfs.FileSystem{tree},
			Prefix: "/browse/" + commit,
		}).ServeHTTP(w, req)
	})
	if os.Getenv("SANDSTORM") != "1" {
		http.ListenAndServe(":8080", r)
	} else {
		uiView := websession.FromHandler(r).
			WithViewInfo(func(p grain_capnp.UiView_getViewInfo) error {
				pogs.Insert(grain_capnp.UiView_ViewInfo_TypeID, p.Results.Struct, viewInfo{
					MatchRequests: []PowerboxDescriptor{{Tags: []Tag{
						{Id: filesystem.Node_TypeID},
						{Id: filesystem.Directory_TypeID},
					}}},
				})
				return nil
			})
		ctx := context.Background()
		_, err := grain.ConnectAPI(ctx, grain_capnp.UiView{
			Client: grain_capnp.UiView_ServerToClient(uiView).Client,
		})
		if err != nil {
			log.Fatal(err)
		}
		<-ctx.Done()
	}
}

type PrefixStripper struct {
	FS     http.FileSystem
	Prefix string
}

func (ps *PrefixStripper) Open(name string) (http.File, error) {
	if !strings.HasPrefix(name, ps.Prefix) {
		panic(name + " does not have expected prefix")
	}
	name = string([]byte(name)[len(ps.Prefix):])
	return ps.FS.Open(name)
}
