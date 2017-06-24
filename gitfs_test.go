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
	ctx := context.TODO()
	dir := &Dir{
		TreeEntry: git.TreeEntry{
			Mode: 0100755,
			Type: "tree",
		},
		g: git.Git{
			os.Getenv("PWD") + "/.git",
		},
	}
	hSlice := make([]byte, len(dir.Hash))
	_, err := fmt.Sscanf(testTree, "%040x", &hSlice)
	if err != nil {
		t.Fatal(err)
	}
	copy(dir.Hash[:], hSlice)

	root := filesystem.Directory_ServerToClient(dir)
	info, err := root.Stat(ctx, func(p filesystem.Node_stat_Params) error {
		return nil
	}).Info().Struct()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Executable() {
		t.Fatal("Git directories should be executable!")
	}
	if info.Writable() {
		t.Fatal("Git objects should be read-only!")
	}
	if info.Which() != filesystem.StatInfo_Which_dir {
		t.Fatal("Wrong type from stat()")
	}

	info, err = root.Walk(ctx, func(p filesystem.Directory_walk_Params) error {
		p.SetName("types.go")
		return nil
	}).Node().Stat(ctx, func(p filesystem.Node_stat_Params) error {
		return nil
	}).Info().Struct()

	if err != nil {
		t.Fatal(err)
	}
	if info.Executable() {
		t.Fatal("types.go should not be executable")
	}
	if info.Writable() {
		t.Fatal("Git objects should be read-only!")
	}
	if info.Which() != filesystem.StatInfo_Which_file {
		t.Fatal("Wrong type from stat()")
	}
	if info.File().Size() != 3327 {
		t.Fatal("Wrong size for types.go:", info.Size())
	}
}
