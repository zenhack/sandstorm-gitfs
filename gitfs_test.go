package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"zenhack.net/go/sandstorm-filesystem/filesystem"
	util_capnp "zenhack.net/go/sandstorm/capnp/util"
	"zenhack.net/go/sandstorm/util"

	"zenhack.net/go/sandstorm-gitfs/git"
)

// part of this repo's history:
const testTree = "247d84d82e4ed81e73661febe5be5952bfd23d10"

func getTreeRoot() filesystem.Directory {
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
		panic(err)
	}
	copy(dir.Hash[:], hSlice)

	return filesystem.Directory_ServerToClient(dir)
}

// Get the root of the test tree, and verify that the StatInfo looks right.
func TestRootDir(t *testing.T) {
	ctx := context.TODO()
	root := getTreeRoot()
	info, err := root.Stat(ctx, func(p filesystem.Node_stat_Params) error {
		return nil
	}).Info().Struct()
	if err != nil {
		t.Fatal(err)
	}
	checkDirStatInfo(t, info)
}

// Validate the StatInfo for types.go in the test tree.
func TestRegularFile(t *testing.T) {
	root := getTreeRoot()
	ctx := context.TODO()
	info, err := root.Walk(ctx, func(p filesystem.Directory_walk_Params) error {
		p.SetName("types.go")
		return nil
	}).Node().Stat(ctx, func(p filesystem.Node_stat_Params) error {
		return nil
	}).Info().Struct()

	if err != nil {
		t.Fatal(err)
	}
	checkFileStatInfo(t, info, false, 3327)
}

// Validate the StatInfo for the `git`subdir in the test tree.
func TestSubDir(t *testing.T) {
	ctx := context.TODO()
	root := getTreeRoot()
	info, err := root.Walk(ctx, func(p filesystem.Directory_walk_Params) error {
		p.SetName("git")
		return nil
	}).Node().Stat(ctx, func(p filesystem.Node_stat_Params) error {
		return nil
	}).Info().Struct()
	if err != nil {
		t.Fatal(err)
	}
	checkDirStatInfo(t, info)
}

// Validate the contents of /types.go in the test tree.
func TestContents(t *testing.T) {
	// We've saved the old contents as testdata/types.go-frozen for comparison:
	buf, err := ioutil.ReadFile("testdata/types.go-frozen")
	expected := string(buf)
	if err != nil {
		t.Fatal(err)
	}
	r, w := io.Pipe()
	bs := util_capnp.ByteStream_ServerToClient(&util.WriteCloserByteStream{WC: w})

	ctx := context.TODO()
	root := getTreeRoot()
	file := filesystem.File{
		Client: root.Walk(ctx, func(p filesystem.Directory_walk_Params) error {
			p.SetName("types.go")
			return nil
		}).Node().Client,
	}

	result := file.Read(ctx, func(p filesystem.File_read_Params) error {
		p.SetSink(bs)
		return nil
	})

	buf, err = ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the call to return:
	_, err = result.Struct()
	if err != nil {
		t.Fatal(err)
	}

	actual := string(buf)
	if actual != expected {
		t.Fatal("Unexpected output: %q", actual)
	}
}

// Test walking into a file that doesn't exist.
func TestAbsentFile(t *testing.T) {
	ctx := context.TODO()
	root := getTreeRoot()

	_, err := root.Walk(ctx, func(p filesystem.Directory_walk_Params) error {
		p.SetName("absent-file")
		return nil
	}).Struct()
	if err == nil {
		t.Fatal("Walk(\"absent-file\") succeeded; should have failed.")
	} else if err != NoSuchFileError {
		t.Fatal(err)
	}
}

// Test listing the root directory.
func TestList(t *testing.T) {
	stream := &testStream{ents: []filesystem.Directory_Entry{}}
	client := filesystem.Directory_Entry_Stream_ServerToClient(stream)

	root := getTreeRoot()
	ctx := context.TODO()
	_, err := root.List(ctx, func(p filesystem.Directory_list_Params) error {
		p.SetStream(client)
		return nil
	}).Struct()
	if err != nil {
		t.Fatal(err)
	}

	if len(stream.ents) != 2 {
		t.Fatalf("Expected 2 results, but got %d.", len(stream.ents))
	}

	for i, expected := range []string{"git", "types.go"} {
		actual, err := stream.ents[i].Name()
		if err != nil {
			t.Fatal(err)
		}
		if actual != expected {
			t.Fatalf("Dirent %d: expected name %q but got %q", i, expected, actual)
		}
	}
	info, err := stream.ents[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	checkDirStatInfo(t, info)
	info, err = stream.ents[1].Info()
	if err != nil {
		t.Fatal(err)
	}
	checkFileStatInfo(t, info, false, 3327)
}

type testStream struct {
	ents []filesystem.Directory_Entry
}

func (t *testStream) Push(p filesystem.Directory_Entry_Stream_push) error {
	ents, err := p.Params.Entries()
	if err != nil {
		return err
	}
	for i := 0; i < ents.Len(); i++ {
		t.ents = append(t.ents, ents.At(i))
	}
	return nil
}

func (t *testStream) Done(p filesystem.Directory_Entry_Stream_done) error {
	return nil
}

func checkDirStatInfo(t *testing.T, info filesystem.StatInfo) {
	if !info.Executable() {
		t.Fatal("Git directories should be executable!")
	}
	if info.Writable() {
		t.Fatal("Git objects should be read-only!")
	}
	if info.Which() != filesystem.StatInfo_Which_dir {
		t.Fatal("Wrong type from stat()")
	}
}

func checkFileStatInfo(t *testing.T, info filesystem.StatInfo, executable bool, size int64) {
	if info.Executable() != executable {
		t.Fatal("File execute bit is wrong")
	}
	if info.Writable() {
		t.Fatal("Git objects should be read-only!")
	}
	if info.Which() != filesystem.StatInfo_Which_file {
		t.Fatal("Wrong type from stat()")
	}
	if info.File().Size() != size {
		t.Fatalf("Wrong size for file; expected %d but got %d.", size, info.Size())
	}
}
