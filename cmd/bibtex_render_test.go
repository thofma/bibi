package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/phd"
	"github.com/thofma/bibi/lib/zb"
)

// This optional integration check exercises the actual output formatter and
// standard BibTeX case conversion, beyond a parser-only round trip.
func TestBibTeXRendering(t *testing.T) {
	if os.Getenv("BIBI_TEST_TEX") != "1" {
		t.Skip("set BIBI_TEST_TEX=1 to run the BibTeX/LaTeX rendering check")
	}
	for _, tool := range []string{"pdflatex", "bibtex"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("rendering check requires %s: %v", tool, err)
		}
	}
	item := zb.Item{
		ID: 1, DocumentType: zb.DocumentType{Code: "j"}, Year: "1999",
		Title:        zb.Title{Title: `Galois groups of \(GL_2(K)\) & 100% results`},
		Contributors: zb.Contributors{Authors: []zb.Author{{Name: "Brinch Hansen, Per"}, {Name: `G{\"o}del, Kurt`}, {Name: "Ducas, Léo"}}},
		Source:       zb.Source{Pages: "1-9", Series: []zb.Series{{Title: "Algebra & Number Theory", ShortTitle: "Alg. Number Theory", Issue: "3", Volume: "42"}}},
	}
	command := &cobra.Command{}
	command.Flags().String("journal", "full", "")
	var output, warnings bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&warnings)
	for _, title := range []string{item.Title.Title, `\LaTeX and Galois theory`} {
		item.Title.Title = title
		entry, err := zb.ItemToBibEntry(item)
		if err != nil {
			t.Fatal(err)
		}
		if err := writeBibTeX(command, entry, zb.ItemJournalNames(item)); err != nil {
			t.Fatal(err)
		}
		item.ID++
	}
	thesis := phd.CreateBibEntryForThesis("Doe, Jane", "2000", `ABC, Galois groups & \(GL_2\)`, "School of Algebra & Geometry")
	if err := writeBibTeX(command, thesis, bibliography.JournalNames{}); err != nil {
		t.Fatal(err)
	}
	if warnings.Len() != 0 {
		t.Fatalf("complete entries produced warnings: %s", warnings.String())
	}
	dir := t.TempDir()
	const document = `\documentclass{article}
\begin{document}
\nocite{*}
\bibliographystyle{plain}
\bibliography{quality}
\end{document}
`
	for name, contents := range map[string]string{"quality.bib": output.String(), "quality.tex": document} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"pdflatex", "-interaction=nonstopmode", "-halt-on-error", "quality.tex"},
		{"bibtex", "quality"},
		{"pdflatex", "-interaction=nonstopmode", "-halt-on-error", "quality.tex"},
	} {
		process := exec.Command(args[0], args[1:]...)
		process.Dir = dir
		if log, err := process.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v\n%s", args[0], err, log)
		}
	}
	bbl, err := os.ReadFile(filepath.Join(dir, "quality.bbl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`Galois groups of \(GL_2(K)\) \& 100\% results`, `\LaTeX and Galois theory`, `Brinch~Hansen`, `G{\"o}del`, `Léo`, `Algebra \& Number Theory`, `School of Algebra \& Geometry`} {
		if !strings.Contains(string(bbl), want) {
			t.Errorf("typeset bibliography is missing %q:\n%s", want, bbl)
		}
	}
}
