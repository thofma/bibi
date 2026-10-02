package bibfile

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Locks live in the OS temporary directory, keeping bibliography directories
// free of sidecars. Advisory locks are released automatically after a crash.
func lockTarget(ctx context.Context, target string) (func(), error) {
	name := filepath.Join(os.TempDir(), fmt.Sprintf("bibi-%x.lock", sha256.Sum256([]byte(target))))
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return nil, fmt.Errorf("open bibliography lock: %w", err)
		}
		for {
			locked, err := tryLock(file)
			if err != nil {
				file.Close()
				return nil, fmt.Errorf("lock bibliography: %w", err)
			}
			if locked {
				break
			}
			timer := time.NewTimer(25 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				file.Close()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
		// A waiting process may have opened a lock removed by its previous owner.
		// Only a lock still attached to the shared pathname can guard a write.
		opened, firstErr := file.Stat()
		current, secondErr := os.Lstat(name)
		if firstErr == nil && secondErr == nil && current.Mode().IsRegular() && os.SameFile(opened, current) {
			return func() { releaseLock(file, name) }, nil
		}
		releaseLock(file, "")
		if firstErr != nil || secondErr != nil && !os.IsNotExist(secondErr) || secondErr == nil && !current.Mode().IsRegular() {
			return nil, fmt.Errorf("bibliography lock is not a readable regular file")
		}
	}
}
