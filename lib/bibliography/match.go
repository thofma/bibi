package bibliography

import "fmt"

// MatchStatus describes identifier evidence, independently of metadata similarity.
type MatchStatus string

const (
	MatchVerified   MatchStatus = "verified"
	MatchUnverified MatchStatus = "unverified"
	MatchConflict   MatchStatus = "conflict"
)

// MatchAssessment explains whether identifiers establish the candidate's identity.
// VerifiedBy is "doi" or the provider whose identifier verified the match.
type MatchAssessment struct {
	Status     MatchStatus
	Reason     string
	VerifiedBy string
}

func (assessment MatchAssessment) Label() string {
	switch assessment.Status {
	case MatchVerified:
		if assessment.VerifiedBy == "doi" {
			return "Verified by DOI"
		}
		return "Verified by " + assessment.VerifiedBy + " identifier"
	case MatchConflict:
		return "Conflicting identifiers"
	default:
		return "Identity unverified"
	}
}

// AssessMatch never treats author, title or year similarity as proof of identity.
// A conflicting DOI vetoes even a shared provider identifier.
func AssessMatch(work, candidate Work, provider string) MatchAssessment {
	doi, candidateDOI := NormalizeDOI(work.DOI), NormalizeDOI(candidate.DOI)
	if doi != "" && candidateDOI != "" {
		if doi == candidateDOI {
			return MatchAssessment{Status: MatchVerified, VerifiedBy: "doi", Reason: fmt.Sprintf("same normalized DOI %q", doi)}
		}
		return MatchAssessment{Status: MatchConflict, Reason: fmt.Sprintf("conflicting DOI: selected=%q candidate=%q", doi, candidateDOI)}
	}
	if id := work.IDs[provider]; id != "" && id == candidate.IDs[provider] {
		return MatchAssessment{Status: MatchVerified, VerifiedBy: provider, Reason: fmt.Sprintf("same %s identifier %q", provider, id)}
	}
	if doi == "" && candidateDOI == "" {
		return MatchAssessment{Status: MatchUnverified, Reason: fmt.Sprintf("neither record has a DOI and no shared %s identifier; identity is unverified", provider)}
	}
	if doi == "" {
		return MatchAssessment{Status: MatchUnverified, Reason: "the discovered work has no DOI; the candidate's DOI cannot establish identity"}
	}
	return MatchAssessment{Status: MatchUnverified, Reason: "the provider candidate has no DOI; identity is unverified"}
}
