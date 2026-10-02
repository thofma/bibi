package bibliography

import "testing"

func TestEscapeTeXTextPreservesNotationAndNames(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"Brinch Hansen, Per and de la Vallée Poussin, Charles", "Brinch Hansen, Per and de la Vallée Poussin, Charles"},
		{`Gödel, Kurt and G{\"o}del, Kurt and Ducas, L\'eo`, `Gödel, Kurt and G{\"o}del, Kurt and Ducas, L\'eo`},
		{`Algebra & Geometry: 50% of A_B #1`, `Algebra \& Geometry: 50\% of A\_B \#1`},
		{`Already \& escaped \% text \_`, `Already \& escaped \% text \_`},
		{`On $GL_2(\mathbb{Q})$ & $K^\times$`, `On $GL_2(\mathbb{Q})$ \& $K^\times$`},
		{`On \(GL_2(K)\) and \[x^2\]`, `On \(GL_2(K)\) and \[x^2\]`},
		{`$$x_1 & x_2$$ & text`, `$$x_1 & x_2$$ \& text`},
		{`\LaTeX, \operatorname{GL} and \url{https://example.com/a_b}`, `\LaTeX, \operatorname{GL} and \url{https://example.com/a_b}`},
		{`A~B ^ C`, `A\textasciitilde{}B \textasciicircum{} C`},
		{"First line\n\tGalois theory", "First line\n\tGalois theory"},
	} {
		t.Run(test.input, func(t *testing.T) {
			if got := EscapeTeXText(test.input); got != test.want {
				t.Errorf("text = %q, want %q", got, test.want)
			}
			if got := EscapeTeXText(test.want); got != test.want {
				t.Errorf("re-encoding changed existing TeX: %q", got)
			}
		})
	}
}

func TestProtectTitle(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"", ""},
		{`Galois theory and $GL_2(K)$`, `{Galois theory and $GL_2(K)$}`},
		{`The {ABC} theorem`, `{The {ABC} theorem}`},
		{`{Galois theory}`, `{Galois theory}`},
		{`\LaTeX and Galois theory`, `{{\LaTeX and Galois theory}}`},
		{`{\LaTeX and Galois theory}`, `{{\LaTeX and Galois theory}}`},
	} {
		if got := ProtectTitle(test.input); got != test.want {
			t.Errorf("ProtectTitle(%q) = %q, want %q", test.input, got, test.want)
		}
		if got := ProtectTitle(test.want); got != test.want {
			t.Errorf("repeated protection changed %q to %q", test.want, got)
		}
	}
}
