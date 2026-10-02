package bibfile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Result describes an append or an identifier-verified duplicate.
type Result struct {
	Key          string
	ExistingKeys []string
	Created      bool
	Entry        string
}

type snapshot struct {
	data []byte
	info os.FileInfo // nil when the file does not yet exist
}

// ValidateTarget checks the destination before an interactive network lookup.
// Add checks it again against its latest contents when saving.
func ValidateTarget(path string, writing bool) error {
	resolved, err := resolveTarget(path)
	if err != nil {
		return err
	}
	_, _, err = inspect(resolved, writing)
	return err
}

// Add preserves every original byte and uses only identifiers in the exported
// entry to detect duplicates. Dry runs create neither a file nor a lock.
func Add(ctx context.Context, path, entry string, dryRun bool) (Result, error) {
	incoming, err := indexFile([]byte(entry))
	if err != nil {
		return Result{}, fmt.Errorf("invalid new BibTeX entry: %w", err)
	}
	if len(incoming.entries) != 1 {
		return Result{}, fmt.Errorf("expected exactly one new BibTeX entry")
	}
	resolved, err := resolveTarget(path)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if !dryRun {
		release, err := lockTarget(ctx, resolved)
		if err != nil {
			return Result{}, err
		}
		defer release()
	}
	before, index, err := inspect(resolved, !dryRun)
	if err != nil {
		return Result{}, err
	}
	keys, err := duplicateKeys(index, incoming.entries[0])
	if err != nil {
		return Result{}, err
	}
	result := Result{Key: incoming.entries[0].key, ExistingKeys: keys, Created: before.info == nil, Entry: entry}
	if len(keys) > 0 || dryRun {
		return result, ctx.Err()
	}
	if err := replace(ctx, resolved, before, appendEntry(before.data, []byte(entry))); err != nil {
		return Result{}, err
	}
	return result, nil
}

func resolveTarget(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve bibliography path: %w", err)
	}
	if _, err := os.Lstat(path); err == nil {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", fmt.Errorf("resolve bibliography symlink: %w", err)
		}
		return resolved, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read bibliography path: %w", err)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", fmt.Errorf("bibliography parent directory must exist: %w", err)
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

func readSnapshot(path string) (snapshot, error) {
	linkInfo, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return snapshot{}, nil
	}
	if err != nil {
		return snapshot{}, fmt.Errorf("read bibliography path: %w", err)
	}
	if !linkInfo.Mode().IsRegular() {
		return snapshot{}, fmt.Errorf("bibliography must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return snapshot{}, fmt.Errorf("read bibliography: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return snapshot{}, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(linkInfo, info) {
		return snapshot{}, fmt.Errorf("bibliography must be a regular file")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return snapshot{}, fmt.Errorf("read bibliography: %w", err)
	}
	return snapshot{data: data, info: info}, nil
}

func inspect(path string, writing bool) (snapshot, fileIndex, error) {
	before, err := readSnapshot(path)
	if err != nil {
		return snapshot{}, fileIndex{}, err
	}
	if writing && before.info != nil && before.info.Mode().Perm()&0222 == 0 {
		return snapshot{}, fileIndex{}, fmt.Errorf("bibliography is read-only")
	}
	index, err := indexFile(before.data)
	if err != nil {
		return snapshot{}, fileIndex{}, fmt.Errorf("cannot safely index bibliography: %w", err)
	}
	return before, index, nil
}

func appendEntry(original, entry []byte) []byte {
	newline := []byte("\n")
	if first := bytes.IndexByte(original, '\n'); first > 0 && original[first-1] == '\r' {
		newline = []byte("\r\n")
	}
	entry = bytes.ReplaceAll(entry, []byte("\r\n"), []byte("\n"))
	if len(newline) == 2 {
		entry = bytes.ReplaceAll(entry, []byte("\n"), newline)
	}
	result := append([]byte(nil), original...)
	if len(result) > 0 {
		if !bytes.HasSuffix(result, newline) {
			result = append(result, newline...)
		}
		if !bytes.HasSuffix(result, append(append([]byte(nil), newline...), newline...)) {
			result = append(result, newline...)
		}
	}
	result = append(result, entry...)
	if !bytes.HasSuffix(result, newline) {
		result = append(result, newline...)
	}
	return result
}

// replace writes and flushes a sibling temporary file, checks for outside edits,
// and only then replaces the destination. The original survives any earlier error.
func replace(ctx context.Context, path string, before snapshot, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".bibi-*.tmp")
	if err != nil {
		return fmt.Errorf("create bibliography temporary file: %w", err)
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	mode := os.FileMode(0600)
	if before.info != nil {
		mode = before.info.Mode().Perm()
	}
	if err := temp.Chmod(mode); err != nil {
		return fmt.Errorf("preserve bibliography permissions: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("write bibliography temporary file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("flush bibliography: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close bibliography temporary file: %w", err)
	}
	if err := unchanged(path, before); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if before.info == nil {
		// An exclusive link also protects a newly created file from another
		// program creating the destination between the check and this operation.
		if err := os.Link(temp.Name(), path); err != nil {
			return fmt.Errorf("create bibliography without overwriting another file: %w", err)
		}
		return nil
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		return fmt.Errorf("replace bibliography: %w", err)
	}
	return nil
}

func unchanged(path string, before snapshot) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) && before.info == nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("bibliography changed while saving; retry: %w", err)
	}
	if before.info == nil || !info.Mode().IsRegular() || !os.SameFile(before.info, info) ||
		info.Mode() != before.info.Mode() || info.Size() != before.info.Size() || !info.ModTime().Equal(before.info.ModTime()) {
		return fmt.Errorf("bibliography changed while saving; retry")
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("check bibliography before saving: %w", err)
	}
	if !bytes.Equal(before.data, current) {
		return fmt.Errorf("bibliography changed while saving; retry")
	}
	return nil
}
