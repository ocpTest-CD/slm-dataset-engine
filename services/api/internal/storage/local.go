package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

type Local struct {
	root string
}

type SavedFile struct {
	Path string
	Hash string
	Size int64
}

func NewLocal(root string) *Local {
	return &Local{root: root}
}

func (s *Local) SaveSource(projectID, filename string, reader io.Reader) (SavedFile, error) {
	dir := filepath.Join(s.root, "sources", projectID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SavedFile{}, err
	}

	tmp, err := os.CreateTemp(dir, "upload-*")
	if err != nil {
		return SavedFile{}, err
	}
	defer tmp.Close()

	hasher := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, hasher), reader)
	if err != nil {
		return SavedFile{}, err
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	target := filepath.Join(dir, hash+"-"+filepath.Base(filename))
	if err := os.Rename(tmp.Name(), target); err != nil {
		return SavedFile{}, err
	}

	return SavedFile{Path: target, Hash: hash, Size: size}, nil
}

func (s *Local) ExportPath(projectID, versionID string) string {
	return filepath.Join(s.root, "exports", projectID, versionID)
}

func (s *Local) InvocationArtifactPath(projectID, invocationID string) string {
	return filepath.Join(s.root, "mcp-artifacts", projectID, invocationID)
}
