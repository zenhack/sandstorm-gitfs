package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"zenhack.net/go/sandstorm-filesystem/filesystem"

	"zenhack.net/go/sandstorm-gitfs/git"
)

// part of this repo's history:
const testTree = "247d84d82e4ed81e73661febe5be5952bfd23d10"

func TestRootDir(t *testing.T) {
	dir := &Dir{
		TreeEntry: git.TreeEntry{
			Mode: 0100755,
			Type: "tree",
		},
		g: git.Git{
			os.Getenv("PWD"),
		},
	}
	hash := dir.Hash[:]
	_, err := fmt.Sscanf(testTree, "%040x", &hash)
	if err != nil {
		t.Fatal(err)
	}
	root := filesystem.Directory_ServerToClient(dir)
	info, err := root.Stat(context.TODO(), func(p filesystem.Node_stat_Params) error {
		return nil
	}).Info().Struct()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Executable() {
		t.Fatal("Git directories should be executable!")
	}
	if info.Writable() {
		t.Fatal("Git objecst should be read-only!")
	}
	if info.Which() != filesystem.StatInfo_Which_dir {
		t.Fatal("Wrong type from stat()")
	}
}
