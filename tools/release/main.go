// Command release validates the artifacts users download, without contacting
// citation services. It is also used by the GoReleaser completion hook.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	archivepath "path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type target struct{ os, arch string }

// Keep the existing eight release targets, including 32-bit Linux and Windows.
var targets = []target{{"linux", "amd64"}, {"linux", "arm64"}, {"linux", "386"},
	{"darwin", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"}, {"windows", "arm64"}, {"windows", "386"}}

var completionFiles = map[string]string{"bash": "bibi.bash", "zsh": "bibi.zsh", "fish": "bibi.fish", "powershell": "bibi.ps1"}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 1 && args[0] == "completions" {
		return generateCompletions()
	}
	if len(args) == 1 && args[0] == "install" {
		return checkGoInstall()
	}
	if len(args) == 2 && args[0] == "verify" {
		_, err := verifyDist(args[1])
		return err
	}
	if len(args) == 3 && args[0] == "formula" {
		version, err := verifyDist(args[1])
		if err != nil {
			return err
		}
		return generateFormula(args[1], version, args[2])
	}
	if len(args) >= 4 && len(args) <= 6 && args[0] == "smoke" {
		version, err := verifyDist(args[1])
		if err != nil {
			return err
		}
		if len(args) >= 5 && strings.TrimPrefix(args[4], "v") != version {
			return fmt.Errorf("archive version %q does not match tag %q", version, args[4])
		}
		commit := ""
		if len(args) == 6 {
			commit = args[5]
		}
		return smokeArchive(args[1], target{args[2], args[3]}, version, commit)
	}
	return fmt.Errorf("usage: go run ./tools/release completions | install | verify DIST | formula DIST COMMIT | smoke DIST OS ARCH [VERSION [COMMIT]]")
}

func archiveName(version string, t target) string {
	osName := map[string]string{"linux": "Linux", "darwin": "Darwin", "windows": "Windows"}[t.os]
	archName := map[string]string{"amd64": "x86_64", "arm64": "arm64", "386": "i386"}[t.arch]
	format := ".tar.gz"
	if t.os == "windows" {
		format = ".zip"
	}
	return "bibi_" + version + "_" + osName + "_" + archName + format
}

func verifyDist(dir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "bibi_*_checksums.txt"))
	if err != nil || len(files) != 1 {
		return "", fmt.Errorf("expected one checksum file in %s, found %d: %v", dir, len(files), err)
	}
	version := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(files[0]), "bibi_"), "_checksums.txt")
	data, err := os.ReadFile(files[0])
	if err != nil {
		return "", err
	}
	sums := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || len(fields[0]) != 64 {
			return "", fmt.Errorf("malformed checksum line: %q", line)
		}
		name := strings.TrimPrefix(fields[1], "*")
		if _, duplicate := sums[name]; duplicate {
			return "", fmt.Errorf("duplicate checksum for %s", name)
		}
		sums[name] = fields[0]
	}
	for _, t := range targets {
		name := archiveName(version, t)
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return "", err
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != sums[name] {
			return "", fmt.Errorf("checksum mismatch for %s", name)
		}
		contents, err := readArchive(filepath.Join(dir, name))
		if err != nil {
			return "", err
		}
		if err := checkContents(contents, t); err != nil {
			return "", fmt.Errorf("%s: %w", name, err)
		}
	}
	name := "bibi_" + version + "_source.tar.gz"
	data, err = os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != sums[name] {
		return "", fmt.Errorf("checksum mismatch for %s", name)
	}
	contents, err := readArchive(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	for _, file := range []string{"go.mod", "go.sum", "main.go", "LICENSE", "internal/journals/README.md", "internal/journals/catalog.json.gz"} {
		if len(strings.TrimSpace(string(contents["bibi_"+version+"/"+file]))) == 0 {
			return "", fmt.Errorf("source archive: missing or empty %s", file)
		}
	}
	fmt.Printf("Verified %d binary archives and the source archive for %s\n", len(targets), version)
	return version, nil
}

func generateFormula(dir, version, commit string) error {
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+([-][0-9A-Za-z.-]+)?$`).MatchString(version) ||
		!regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(commit) {
		return fmt.Errorf("invalid formula version or commit")
	}
	name := "bibi_" + version + "_source.tar.gz"
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	formula := fmt.Sprintf(`class Bibi < Formula
  desc "Retrieve BibTeX for mathematical literature"
  homepage "https://github.com/thofma/bibi"
  license "MIT"
  url "https://github.com/thofma/bibi/releases/download/v%s/%s"
  version %q
  sha256 "%x"
  depends_on "go" => :build

  def install
    ENV["CGO_ENABLED"] = "0"
    ldflags = "-s -w -X github.com/thofma/bibi/internal/buildinfo.Version=#{version} -X github.com/thofma/bibi/internal/buildinfo.Commit=%s"
    system "go", "build", "-trimpath", "-mod=readonly", "-ldflags", ldflags, "-o", bin/"bibi", "."
    generate_completions_from_executable(bin/"bibi", "completion", shell_parameter_format: :cobra)
    pkgshare.install "internal/journals/README.md" => "journals.md"
  end

  test do
    assert_match "bibi v#{version}", shell_output("#{bin}/bibi --version")
    assert_equal "Invent. Math.\n", shell_output("#{bin}/bibi abbr inventiones mathematicae")
  end
end
`, version, name, version, sha256.Sum256(data), commit)
	return os.WriteFile(filepath.Join(dir, "bibi.rb"), []byte(formula), 0644)
}

func readArchive(path string) (map[string][]byte, error) {
	contents := make(map[string][]byte)
	add := func(name string, reader io.Reader) error {
		if archivepath.IsAbs(name) || strings.ContainsAny(name, "\\:") || name == ".." || strings.HasPrefix(name, "../") || archivepath.Clean(name) != name {
			return fmt.Errorf("unexpected archive path %q", name)
		}
		if _, duplicate := contents[name]; duplicate {
			return fmt.Errorf("duplicate archive member %q", name)
		}
		data, err := io.ReadAll(io.LimitReader(reader, 64<<20))
		if err != nil {
			return err
		}
		if len(data) >= 64<<20 {
			return fmt.Errorf("archive member %q exceeds size limit", name)
		}
		contents[name] = data
		return nil
	}
	if strings.HasSuffix(path, ".zip") {
		archive, err := zip.OpenReader(path)
		if err != nil {
			return nil, err
		}
		defer archive.Close()
		for _, member := range archive.File {
			if member.FileInfo().IsDir() {
				continue
			}
			if !member.Mode().IsRegular() {
				return nil, fmt.Errorf("unexpected archive member %q", member.Name)
			}
			reader, err := member.Open()
			if err != nil {
				return nil, err
			}
			err = add(member.Name, reader)
			reader.Close()
			if err != nil {
				return nil, err
			}
		}
		return contents, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer compressed.Close()
	archive := tar.NewReader(compressed)
	for {
		member, err := archive.Next()
		if err == io.EOF {
			return contents, nil
		}
		if err != nil {
			return nil, err
		}
		if member.Typeflag == tar.TypeDir || member.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if member.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("unexpected archive member %q", member.Name)
		}
		if err := add(member.Name, archive); err != nil {
			return nil, err
		}
	}
}

func binaryName(goos string) string {
	if goos == "windows" {
		return "bibi.exe"
	}
	return "bibi"
}

func checkContents(contents map[string][]byte, t target) error {
	required := []string{binaryName(t.os), "README.md", "LICENSE", "data/journals.md"}
	for _, name := range completionFiles {
		required = append(required, "completions/"+name)
	}
	for _, name := range required {
		if len(strings.TrimSpace(string(contents[name]))) == 0 {
			return fmt.Errorf("missing or empty %s", name)
		}
	}
	return nil
}

func smokeArchive(dir string, t target, version, commit string) error {
	if t.os != runtime.GOOS || (t.arch != runtime.GOARCH && !(t.arch == "386" && runtime.GOARCH == "amd64")) {
		return fmt.Errorf("cannot run %s/%s on %s/%s", t.os, t.arch, runtime.GOOS, runtime.GOARCH)
	}
	contents, err := readArchive(filepath.Join(dir, archiveName(version, t)))
	if err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "bibi-archive-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	for name, data := range contents {
		path := filepath.Join(temp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0755); err != nil {
			return err
		}
	}
	if err := checkExecutable(filepath.Join(temp, binaryName(t.os)), version, commit); err != nil {
		return err
	}
	fmt.Printf("Packaged %s/%s binary passed offline checks\n", t.os, t.arch)
	return nil
}

func checkExecutable(path, version, commit string) error {
	output, err := invoke(path, "--version")
	if err != nil {
		return err
	}
	expected := "bibi " + version
	if version != "devel" {
		expected = "bibi v" + strings.TrimPrefix(version, "v")
	}
	if strings.TrimSpace(output) != expected && !strings.HasPrefix(output, expected+" (commit ") {
		return fmt.Errorf("unexpected version output %q, expected %q", output, expected)
	}
	if commit != "" && !strings.Contains(output, "commit "+commit[:min(12, len(commit))]) {
		return fmt.Errorf("version output %q does not identify commit %s", output, commit)
	}
	if output, err = invoke(path, "--help"); err != nil || !strings.Contains(output, "--version") {
		return fmt.Errorf("help smoke check failed: %v; output=%q", err, output)
	}
	if output, err = invoke(path, "abbr", "inventiones", "mathematicae"); err != nil || output != "Invent. Math.\n" {
		return fmt.Errorf("offline abbreviation smoke check failed: %v; output=%q", err, output)
	}
	for shell := range completionFiles {
		if output, err = invoke(path, "completion", shell); err != nil || !strings.Contains(output, "bibi") {
			return fmt.Errorf("%s completion smoke check failed: %v; output=%q", shell, err, output)
		}
	}
	return nil
}

func invoke(path string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, args...)
	command.Dir = filepath.Dir(path) // Never depend on the source checkout.
	var stderr strings.Builder
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil || stderr.Len() != 0 {
		return "", fmt.Errorf("%s %v: %v; stderr=%s", path, args, err, stderr.String())
	}
	return string(output), nil
}

func generateCompletions() error {
	temp, err := os.MkdirTemp("", "bibi-completions-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	path := filepath.Join(temp, binaryName(runtime.GOOS))
	command := exec.Command("go", "build", "-o", path, ".")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("build completion generator: %w\n%s", err, output)
	}
	if err := os.MkdirAll(".release/completions", 0755); err != nil {
		return err
	}
	for shell, name := range completionFiles {
		output, err := invoke(path, "completion", shell)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(".release/completions", name), []byte(output), 0644); err != nil {
			return err
		}
	}
	return nil
}

func checkGoInstall() error {
	temp, err := os.MkdirTemp("", "bibi-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	command := exec.Command("go", "install", ".")
	command.Env = append(os.Environ(), "GOBIN="+temp)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("go install: %w\n%s", err, output)
	}
	return checkExecutable(filepath.Join(temp, binaryName(runtime.GOOS)), "devel", "")
}
