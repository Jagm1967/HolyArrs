package backups

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// MirrorEnv names an optional folder where every created backup zip is copied
// atomically (for example a mounted iCloud Drive or NAS backup directory).
const MirrorEnv = "HOLYARRS_BACKUP_DIR"

func mirrorDir() string {
	return strings.TrimSpace(os.Getenv(MirrorEnv))
}

// mirrorCopy writes a complete backup zip into HOLYARRS_BACKUP_DIR using a
// temporary .part file followed by rename so cloud sync clients never upload
// a partially written file. Mirror errors are logged without failing the
// primary backup inside /config/backups.
func mirrorCopy(srcPath, name string) {
	dir := mirrorDir()
	if dir == "" {
		return
	}
	if err := copyAtomic(srcPath, dir, name); err != nil {
		log.Printf("backup: mirror %s to %s: %v", name, dir, err)
	}
}

func copyAtomic(srcPath, dir, name string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	final := filepath.Join(dir, name)
	part := final + ".part"
	dst, err := os.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			_ = dst.Close()
			_ = os.Remove(part)
		}
	}()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	if err := dst.Sync(); err != nil {
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	if err := os.Rename(part, final); err != nil {
		return err
	}
	ok = true
	return nil
}

// mirrorRemove deletes a pruned backup zip from HOLYARRS_BACKUP_DIR so cloud
// storage follows the same retention policy as /config/backups.
func mirrorRemove(name string) {
	dir := mirrorDir()
	if dir == "" {
		return
	}
	if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("backup: mirror remove %s: %v", name, err)
	}
}
