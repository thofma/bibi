package bibfile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const newEntry = "@misc{New,\n    title = {Local fields},\n    doi = {10.1000/new},\n}\n"

func TestAppendPreservesOriginalTextAndNewlines(t *testing.T) {
	for _, original := range []string{
		"", "% handwritten notes", "@string{j={My journal}}\n@article{Old, journal=j}\n",
		"% a comment\r\n@misc(Old, title={A {TeX} title})\r\n",
		"@misc{Old, title={Équations}}\n\n",
	} {
		t.Run(fmt.Sprintf("%q", original), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "references.bib")
			if err := os.WriteFile(path, []byte(original), 0640); err != nil {
				t.Fatal(err)
			}
			result, err := Add(context.Background(), path, newEntry, false)
			if err != nil || result.Created || result.Key != "New" {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(data, []byte(original)) || strings.Count(string(data), "@misc{New,") != 1 {
				t.Fatalf("original changed: %q", data)
			}
			if strings.Contains(original, "\r\n") && bytes.Contains(bytes.ReplaceAll(bytes.TrimPrefix(data, []byte(original)), []byte("\r\n"), nil), []byte("\n")) {
				t.Fatalf("new entry did not use CRLF: %q", data)
			}
			if _, err := indexFile(data); err != nil {
				t.Fatal(err)
			}
			info, _ := os.Stat(path)
			if runtime.GOOS != "windows" && info.Mode().Perm() != 0640 {
				t.Fatalf("permissions changed to %v", info.Mode())
			}
			assertNoTemps(t, filepath.Dir(path))
		})
	}
}

func TestCreateDryRunAndDuplicateNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "references.bib")
	result, err := Add(context.Background(), path, newEntry, true)
	if err != nil || !result.Created || result.Entry != newEntry {
		t.Fatalf("dry run result=%+v error=%v", result, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dry run created a file: %v", err)
	}
	assertNoTemps(t, filepath.Dir(path))
	result, err = Add(context.Background(), path, newEntry, false)
	if err != nil || !result.Created {
		t.Fatalf("create result=%+v error=%v", result, err)
	}
	before, _ := os.Stat(path)
	for _, dryRun := range []bool{false, true} {
		result, err = Add(context.Background(), path, strings.Replace(newEntry, "{New,", "{AnotherKey,", 1), dryRun)
		if err != nil || len(result.ExistingKeys) != 1 || result.ExistingKeys[0] != "New" {
			t.Fatalf("duplicate result=%+v error=%v", result, err)
		}
		data, _ := os.ReadFile(path)
		after, _ := os.Stat(path)
		if string(data) != newEntry || !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
			t.Fatal("duplicate modified the file")
		}
	}
}

func TestFailuresPreserveFiles(t *testing.T) {
	for _, original := range []string{`@misc{New}`, `@misc{Broken,title={`, `@misc{Old,doi=undefined}`, `@misc{Old,doi={10.1000/new},MRNUMBER=123}`} {
		t.Run(original, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "references.bib")
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			entry := strings.Replace(newEntry, "    title", "    MRNUMBER = 456,\n    title", 1)
			if _, err := Add(context.Background(), path, entry, false); err == nil {
				t.Fatal("expected an error")
			}
			data, _ := os.ReadFile(path)
			if string(data) != original {
				t.Fatal("failed add changed the file")
			}
			assertNoTemps(t, filepath.Dir(path))
		})
	}
}

func TestDestinationValidationAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	if err := ValidateTarget(dir, true); err == nil {
		t.Fatal("accepted directory")
	}
	if err := ValidateTarget(filepath.Join(dir, "missing", "refs.bib"), true); err == nil {
		t.Fatal("accepted missing parent")
	}
	path := filepath.Join(dir, "real.bib")
	if err := os.WriteFile(path, []byte("% original\n"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTarget(path, true); err == nil {
		t.Fatal("accepted read-only file for writing")
	}
	if err := ValidateTarget(path, false); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "linked.bib")
	if err := os.Symlink("real.bib", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Add(context.Background(), link, newEntry, false); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Lstat(link); info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("replaced the symlink")
	}
	data, _ := os.ReadFile(path)
	if !bytes.HasPrefix(data, []byte("% original\n")) || !bytes.Contains(data, []byte("@misc{New")) {
		t.Fatal("did not append to symlink target")
	}
	broken := filepath.Join(dir, "broken.bib")
	if err := os.Symlink("missing.bib", broken); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTarget(broken, true); err == nil {
		t.Fatal("accepted dangling symlink")
	}
}

func TestOutsideEditsAndCancellationPreventReplacement(t *testing.T) {
	for _, action := range []string{"edit", "replace inode", "created", "deleted", "cancel"} {
		t.Run(action, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "references.bib")
			if action != "created" {
				if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, err := readSnapshot(path)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch action {
			case "edit", "created":
				err = os.WriteFile(path, []byte("external change"), 0600)
			case "replace inode":
				err = os.Remove(path)
				if err == nil {
					err = os.WriteFile(path, []byte("external change"), 0600)
				}
			case "deleted":
				err = os.Remove(path)
			case "cancel":
				cancel()
			}
			if err != nil {
				t.Fatal(err)
			}
			changed, _ := os.ReadFile(path)
			if err := replace(ctx, path, before, []byte("replacement")); err == nil {
				t.Fatal("overwrote a changed file or ignored cancellation")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(changed, after) {
				t.Fatal("failed replace changed file")
			}
			assertNoTemps(t, filepath.Dir(path))
		})
	}
}

func TestConcurrentAddsPreserveEveryEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "references.bib")
	const count = 12
	var wg sync.WaitGroup
	errors := make(chan error, count)
	for i := range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Add(context.Background(), path, fmt.Sprintf("@misc{Key%d,doi={10.1000/work%d}}\n", i, i), false)
			errors <- err
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	index, err := indexFile(data)
	if err != nil || len(index.entries) != count {
		t.Fatalf("entries=%d error=%v", len(index.entries), err)
	}
	assertNoTemps(t, filepath.Dir(path))
}

func TestWaitingForLockCanBeCancelled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "references.bib")
	resolved, err := resolveTarget(path)
	if err != nil {
		t.Fatal(err)
	}
	release, err := lockTarget(context.Background(), resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	if _, err := Add(ctx, path, newEntry, false); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("created file while waiting")
	}
}

func assertNoTemps(t *testing.T, dir string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, ".bibi-*.tmp"))
	if err != nil || len(files) != 0 {
		t.Fatalf("leftover temporary files=%v error=%v", files, err)
	}
}
