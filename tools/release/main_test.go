package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureContents(goos string) map[string][]byte {
	contents := map[string][]byte{binaryName(goos): []byte("fixture executable"), "README.md": []byte("README"),
		"LICENSE": []byte("Fixture license text"), "data/journals.md": []byte("Journal catalog notes")}
	for _, name := range completionFiles {
		contents["completions/"+name] = []byte("fixture shell completion")
	}
	return contents
}

func writeFixtureArchive(t *testing.T, path string, contents map[string][]byte) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if strings.HasSuffix(path, ".zip") {
		archive := zip.NewWriter(file)
		for name, data := range contents {
			writer, err := archive.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write(data); err != nil {
				t.Fatal(err)
			}
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		return
	}
	compressed := gzip.NewWriter(file)
	archive := tar.NewWriter(compressed)
	for name, data := range contents {
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
}

func fixtureDist(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	var sums strings.Builder
	for _, target := range targets {
		name := archiveName("0.5.0", target)
		path := filepath.Join(dir, name)
		writeFixtureArchive(t, path, fixtureContents(target.os))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&sums, "%x  %s\n", sha256.Sum256(data), name)
	}
	source := map[string][]byte{}
	for _, name := range []string{"go.mod", "go.sum", "main.go", "LICENSE", "internal/journals/README.md", "internal/journals/catalog.json.gz"} {
		source["bibi_0.5.0/"+name] = []byte("Source fixture")
	}
	name := "bibi_0.5.0_source.tar.gz"
	path := filepath.Join(dir, name)
	writeFixtureArchive(t, path, source)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(&sums, "%x  %s\n", sha256.Sum256(data), name)
	if err := os.WriteFile(filepath.Join(dir, "bibi_0.5.0_checksums.txt"), []byte(sums.String()), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFormulaMatchesSourceChecksumAndBuild(t *testing.T) {
	dir := fixtureDist(t)
	commit := strings.Repeat("a", 40)
	if err := generateFormula(dir, "0.5.0", commit); err != nil {
		t.Fatal(err)
	}
	formula, err := os.ReadFile(filepath.Join(dir, "bibi.rb"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(dir, "bibi_0.5.0_source.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"class Bibi < Formula", `version "0.5.0"`, fmt.Sprintf(`sha256 "%x"`, sha256.Sum256(source)),
		"releases/download/v0.5.0/bibi_0.5.0_source.tar.gz", "internal/buildinfo.Commit=" + commit,
		"generate_completions_from_executable", "-mod=readonly"} {
		if !strings.Contains(string(formula), want) {
			t.Errorf("formula is missing %q", want)
		}
	}
	if err := generateFormula(dir, `0.5.0"; invalid`, commit); err == nil {
		t.Fatal("unsafe version was accepted")
	}
	if err := generateFormula(dir, "0.5.0", "invalid"); err == nil {
		t.Fatal("invalid commit was accepted")
	}
}

func TestVerifyDistDetectsDamagedAndMissingPackages(t *testing.T) {
	dir := fixtureDist(t)
	if version, err := verifyDist(dir); err != nil || version != "0.5.0" {
		t.Fatalf("complete packages: version=%q error=%v", version, err)
	}
	path := filepath.Join(dir, archiveName("0.5.0", target{"windows", "arm64"}))
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.WriteString("corruption")
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifyDist(dir); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("damaged archive was accepted: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyDist(dir); err == nil {
		t.Fatal("missing architecture was accepted")
	}
}

func TestMissingRequiredPackageContentsFail(t *testing.T) {
	for _, missing := range []string{"LICENSE", "README.md", "completions/bibi.zsh", "data/journals.md", "bibi"} {
		contents := fixtureContents("linux")
		delete(contents, missing)
		if err := checkContents(contents, target{"linux", "amd64"}); err == nil || !strings.Contains(err.Error(), missing) {
			t.Errorf("missing %s was accepted: %v", missing, err)
		}
	}
	contents := fixtureContents("linux")
	contents["LICENSE"] = []byte(" \n\t")
	if err := checkContents(contents, target{"linux", "amd64"}); err == nil {
		t.Fatal("empty license was accepted")
	}
}

func TestArchivesRejectPathsOutsideTheirRoot(t *testing.T) {
	for _, format := range []string{".tar.gz", ".zip"} {
		for _, name := range []string{"../outside", "/absolute", "C:/outside", "directory/../outside", "directory\\outside"} {
			path := filepath.Join(t.TempDir(), "archive"+format)
			writeFixtureArchive(t, path, map[string][]byte{name: []byte("data")})
			if _, err := readArchive(path); err == nil {
				t.Errorf("accepted path %q in %s", name, format)
			}
		}
	}
}
