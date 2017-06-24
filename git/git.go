package git

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Hash [sha1.Size]byte

func (h *Hash) String() string {
	return fmt.Sprintf("%040x", h[:])
}

type TreeEntry struct {
	Mode uint32
	Type string
	Hash Hash
	Name string
}

type Git struct {
	GitDir string
}

func (g *Git) Command(args ...string) *exec.Cmd {
	return exec.Command("git", append([]string{"--git-dir=" + g.GitDir}, args...)...)
}

func (g *Git) GetFileSize(h *Hash) (int64, error) {
	out, err := g.Command("cat-file", "-s", h.String()).Output()
	if err != nil {
		return 0, err
	}
	ret := int64(0)
	_, err = fmt.Sscanf(string(out), "%d", &ret)
	return ret, err
}

func (g *Git) LsTree(h *Hash) ([]TreeEntry, error) {
	out, err := g.Command("ls-tree", h.String()).Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) != 0 && lines[len(lines)-1] == "" {
		// Strip the trailing blank line. This if statement should theoretically always be
		// executed, but we check just in case.
		lines = lines[:len(lines)-1]
	}
	ents := make([]TreeEntry, len(lines))
	for i := range lines {
		hSlice := make([]byte, len(ents[i].Hash))
		_, err = fmt.Sscanf(lines[i], "%o %s %040x\t%s",
			&ents[i].Mode,
			&ents[i].Type,
			&hSlice,
			&ents[i].Name,
		)
		copy(ents[i].Hash[:], hSlice)
		if err != nil {
			return nil, err
		}
	}
	return ents, nil
}

func (g *Git) ReadFile(h *Hash) (io.ReadCloser, error) {
	return g.Command("git", "cat-file", h.String()).StdoutPipe()
}

func InitBare(path string) (*Git, error) {
	err := exec.Command("git", "init", "--bare", path).Run()
	return &Git{GitDir: path}, err
}
