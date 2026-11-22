package backups

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HolyArrs/holyarrs/app/backend/internal/core/database"
)

func TestMirrorBackupDirCreateAndPrune(t *testing.T) {
	dir := t.TempDir()
	mirror := filepath.Join(t.TempDir(), "icloud-backups")
	t.Setenv(MirrorEnv, mirror)

	st, err := database.Open(dir, database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	now := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	m := &Manager{Store: st, Dir: dir, Version: "1.2.0-fork", Now: func() time.Time { return now }}

	oldBackup, err := m.Create(context.Background(), KindScheduled)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(mirror, oldBackup.Name)); err != nil {
		t.Fatalf("mirrored old backup missing: %v", err)
	}

	// Create a newer backup 40 days later and prune with Light (28-day max age).
	now = now.Add(40 * 24 * time.Hour)
	newBackup, err := m.Create(context.Background(), KindScheduled)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Prune(Light); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(mirror, oldBackup.Name)); !os.IsNotExist(err) {
		t.Fatalf("pruned backup should be removed from mirror, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(mirror, newBackup.Name)); err != nil {
		t.Fatalf("newest backup should remain in mirror: %v", err)
	}
}
