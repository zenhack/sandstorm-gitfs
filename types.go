package main

import (
	"errors"
	"io"
	"io/ioutil"
	"sort"

	"zenhack.net/go/sandstorm-filesystem/filesystem"
	"zenhack.net/go/sandstorm/util"

	"zenhack.net/go/sandstorm-gitfs/git"
)

var (
	NoSuchFileError = errors.New("No such file")
)

type Dir struct {
	git.TreeEntry
	g        git.Git
	contents []git.TreeEntry
}

type File struct {
	git.TreeEntry
	g git.Git
}

func (d *Dir) Stat(p filesystem.Node_stat) error {
	info, err := p.Results.NewInfo()
	if err != nil {
		return err
	}
	return statTreeEntry(&d.g, &d.TreeEntry, info)
}

func (d *Dir) List(p filesystem.Directory_list) error {
	err := d.ensureContents()
	if err != nil {
		return err
	}
	stream := p.Params.Stream()
	_, err = stream.Push(p.Ctx, func(p filesystem.Directory_Entry_Stream_push_Params) error {
		ents, err := p.NewEntries(int32(len(d.contents)))
		if err != nil {
			return err
		}
		for i := range d.contents {
			info, err := ents.At(i).NewInfo()
			if err != nil {
				return err
			}
			ents.At(i).SetName(d.contents[i].Name)
			err = statTreeEntry(&d.g, &d.contents[i], info)
			if err != nil {
				return err
			}
		}
		return nil
	}).Struct()
	if err != nil {
		return err
	}
	_, err = stream.Done(p.Ctx, func(p filesystem.Directory_Entry_Stream_done_Params) error {
		return nil
	}).Struct()
	return err
}

func (d *Dir) Walk(p filesystem.Directory_walk) error {
	name, err := p.Params.Name()
	if err != nil {
		return err
	}
	err = d.ensureContents()
	if err != nil {
		return err
	}
	i := sort.Search(len(d.contents), func(i int) bool {
		return d.contents[i].Name >= name
	})
	if i == len(d.contents) {
		return NoSuchFileError
	}
	ent := d.contents[i]
	if ent.Name != name {
		return NoSuchFileError
	}
	switch ent.Type {
	case "tree":
		p.Results.SetNode(filesystem.Node{filesystem.Directory_ServerToClient(&Dir{
			TreeEntry: ent,
			g:         d.g,
		}).Client})
	case "blob":
		p.Results.SetNode(filesystem.Node{filesystem.File_ServerToClient(&File{
			TreeEntry: ent,
			g:         d.g,
		}).Client})
	}
	return nil
}

func (d *Dir) ensureContents() error {
	if d.contents != nil {
		return nil
	}
	ents, err := d.g.LsTree(&d.Hash)
	if err != nil {
		return err
	}
	sort.Slice(ents, func(i, j int) bool {
		return ents[i].Name < ents[j].Name
	})
	d.contents = ents
	return nil
}

func (f *File) Stat(p filesystem.Node_stat) error {
	info, err := p.Results.NewInfo()
	if err != nil {
		return err
	}
	return statTreeEntry(&f.g, &f.TreeEntry, info)
}

func (f *File) Read(p filesystem.File_read) error {
	var src io.Reader

	startAt := int64(p.Params.StartAt())
	amount := int64(p.Params.Amount())

	r, err := f.g.ReadFile(&f.Hash)
	if err != nil {
		return err
	}
	defer r.Close()

	if startAt > 0 {
		prefix := io.LimitReader(r, startAt)
		_, err = io.Copy(ioutil.Discard, prefix)

	}
	if amount > 0 {
		src = io.LimitReader(r, amount)
	} else {
		src = r
	}

	bs := &util.ByteStreamWriteCloser{
		Ctx: p.Ctx,
		Obj: p.Params.Sink(),
	}
	_, err = io.Copy(bs, src)
	if err != nil {
		return err
	}
	return bs.Close()
}

func statTreeEntry(g *git.Git, e *git.TreeEntry, info filesystem.StatInfo) error {
	switch e.Type {
	case "tree":
		info.SetDir()
		info.SetExecutable(true)
	case "blob":
		info.SetFile()
		sz, err := g.GetFileSize(&e.Hash)
		if err != nil {
			return err
		}
		info.File().SetSize(int64(sz))
		info.SetExecutable(e.Mode&0100 != 0)
	}
	info.SetWritable(false)
	return nil
}
