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
	return fmt.Sprintf("%040x", h)
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
	out, err := g.Command("git", "cat-file", "-s", h.String()).Output()
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
	ents := make([]TreeEntry, len(lines))
	for i := range lines {
		_, err = fmt.Sscanf(lines[i], "%o %s %040x    %s",
			&ents[i].Mode,
			&ents[i].Type,
			&ents[i].Hash,
			&ents[i].Name,
		)
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
