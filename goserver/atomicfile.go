package main

// atomicfile.go — writes a private file (0600, in a 0700 directory) by
// writing a temp file and renaming it into place, so a crash mid-write can
// never leave a truncated file behind. Used for everything persisted under
// the data dir, all of which can contain provider credentials.

import (
	"os"
	"path/filepath"
)

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*") // created 0600
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp.Name())
		if werr != nil {
			return werr
		}
		return cerr
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}
