package bibliography

import (
	"strings"
	"testing"
)

func TestAssessmentUsesIdentifiersAndPreservesMatchAPI(t *testing.T) {
	for _, test := range []struct {
		name       string
		work       Work
		candidate  Work
		status     MatchStatus
		verifiedBy string
	}{
		{"normalized DOI", Work{DOI: "doi:10.1000/ABC"}, Work{DOI: "https://doi.org/10.1000/abc"}, MatchVerified, "doi"},
		{"native ID", Work{IDs: map[string]string{"mr": "MR1"}}, Work{IDs: map[string]string{"mr": "MR1"}}, MatchVerified, "mr"},
		{"DOI conflict vetoes native ID", Work{DOI: "10.1000/a", IDs: map[string]string{"mr": "MR1"}}, Work{DOI: "10.1000/b", IDs: map[string]string{"mr": "MR1"}}, MatchConflict, ""},
		{"candidate lacks DOI", Work{DOI: "10.1000/a"}, Work{}, MatchUnverified, ""},
		{"selected lacks DOI", Work{}, Work{DOI: "10.1000/a"}, MatchUnverified, ""},
		{"identical metadata", Work{Title: "Work", Authors: []string{"Doe"}, Year: "2000"}, Work{Title: "Work", Authors: []string{"Doe"}, Year: "2000"}, MatchUnverified, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			assessment := AssessMatch(test.work, test.candidate, "mr")
			if assessment.Status != test.status || assessment.VerifiedBy != test.verifiedBy || assessment.Reason == "" || assessment.Label() == "" {
				t.Fatalf("assessment=%+v", assessment)
			}
			exact, compatible := Match(test.work, test.candidate, "mr")
			if exact != (test.status == MatchVerified) || compatible != (test.status != MatchConflict) || MatchReason(test.work, test.candidate, "mr") != assessment.Reason {
				t.Fatal("legacy match API disagrees with assessment")
			}
		})
	}
}

func TestComparisonShowsEvidenceDifferencesAndMissingMetadata(t *testing.T) {
	work := Work{Title: `Galois groups of $GL_2$`, Authors: []string{"One, Alice", "Two, Bob"},
		Year: "1967", Type: "book", Edition: "First", DOI: "10.1000/book", IDs: map[string]string{"zb": "42"}}
	candidate := Work{Title: `Galois groups of $GL_2$`, Editors: []string{"Editor, Eve"},
		Year: "2006", Type: "book", Edition: "Second", Notes: "Translation of the original edition", DOI: "doi:10.1000/BOOK"}
	details := ComparisonDetails(work, candidate, AssessMatch(work, candidate, "mr"))
	for _, want := range []string{"Match status: Verified by DOI", "same normalized DOI", "Review required:", "supplied editions differ",
		"Differences:", `Year: selected "1967"; provider "2006"`, "Selected work:", "Provider candidate:", "Authors: One, Alice; Two, Bob",
		"Authors: unavailable", "Edition: First", "Edition: Second", "Venue: unavailable", "Editor, Eve", candidate.Notes,
		"zb identifier: unavailable", work.Title} {
		if !strings.Contains(details, want) {
			t.Errorf("missing %q in comparison:\n%s", want, details)
		}
	}
	if strings.Contains(details, "DOI: selected") || candidate.DOI != "doi:10.1000/BOOK" {
		t.Fatal("DOI spelling became a difference or source metadata was changed")
	}
}

func TestReviewReasonsDistinguishVersionsWithoutGuessingFromMetadata(t *testing.T) {
	for _, test := range []struct {
		name            string
		work, candidate Work
		want            string
	}{
		{"edition", Work{Edition: "1"}, Work{Edition: "2"}, "editions differ"},
		{"edition spelling", Work{Edition: " Second "}, Work{Edition: "second"}, ""},
		{"missing edition", Work{Edition: "2"}, Work{}, ""},
		{"preprint to journal", Work{Type: "preprint"}, Work{Type: "journal-article"}, "selected work is a preprint"},
		{"journal to preprint", Work{Type: "article-journal"}, Work{Type: "preprint"}, "candidate is a preprint"},
		{"year difference", Work{Year: "2020"}, Work{Year: "2024"}, ""},
		{"title and link", Work{Title: "Second edition preprint", IDs: map[string]string{"arxiv": "2301.12345"}}, Work{Type: "article"}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			reasons := ReviewReasons(test.work, test.candidate)
			if test.want == "" {
				if len(reasons) != 0 {
					t.Fatalf("inferred version change: %v", reasons)
				}
			} else if !strings.Contains(strings.Join(reasons, "\n"), test.want) {
				t.Fatalf("missing review reason %q in %v", test.want, reasons)
			}
		})
	}
}
