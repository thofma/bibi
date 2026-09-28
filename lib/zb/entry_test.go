package zb

import (
	"testing"
)

func TestParseToStructReturnsJSONError(t *testing.T) {
	response, err := ParseToStruct(`{"result":`)
	if err == nil {
		t.Fatal("ParseToStruct() error = nil, want JSON parse error")
	}
	if len(response.Result) != 0 {
		t.Fatalf("ParseToStruct() result count = %d, want 0 on parse error", len(response.Result))
	}
}

func TestParseToStructDecodesBookYear(t *testing.T) {
	response, err := ParseToStruct(`{
		"result": [{
			"source": {"book": [{"book_id": 7, "publisher": "MSP", "year": "2019"}]}
		}]
	}`)
	if err != nil {
		t.Fatalf("ParseToStruct() error = %v", err)
	}
	if len(response.Result) != 1 || len(response.Result[0].Source.Book) != 1 {
		t.Fatalf("ParseToStruct() decoded unexpected response: %+v", response)
	}
	if got := response.Result[0].Source.Book[0].Year; got != "2019" {
		t.Errorf("Book.Year = %q, want %q", got, "2019")
	}
}

// func TestParse(t *testing.T) {
// 	jsonData := `{"result":[{"biographic_references":[],"contributors":{"authors":[{"aliases":[],"checked":"0","codes":["biasse.jean-francois"],"name":"Biasse, Jean-François"},{"aliases":[],"checked":"0","codes":["fieker.claus"],"name":"Fieker, Claus"}],"author_references":[],"editors":[]},"database":"Zbl","datestamp":"2014-09-05T10:39:52Z","document_type":{"code":"j","description":"journal article"},"editorial_contributions":[{"language":"English","reviewer":{"author_code":null,"reviewer_id":null,"name":null,"sign":null},"text":"Summary: We describe how to compute the ideal class group and the unit group of an order in a number field in subexponential time. Our method relies on the generalized Riemann hypothesis and other usual heuristics concerning the smoothness of ideals. It applies to arbitrary classes of number fields, including those for which the degree goes to infinity.","contribution_type":"summary"}],"id":6340507,"identifier":"1369.11103","keywords":["subexponential time"],"language":{"languages":["English"],"addition":[null]},"license":[],"links":[{"identifier":"10.1112/S1461157014000345","type":"doi","url":"https://doi.org/10.1112/S1461157014000345"}],"msc":[{"code":"11Y40","scheme":"msc2020","text":"Algebraic number theory computations"},{"code":"11R29","scheme":"msc2020","text":"Class numbers, class groups, discriminants"},{"code":"11R27","scheme":"msc2020","text":"Units and factorization"}],"references":[{"doi":"10.1007/978-3-642-14518-6_19","position":"1","text":"DOI: 10.1007/978-3-642-14518-6_19","zbmath":{"author_codes":["jao.david","soukharev.vladimir"],"document_id":5793682,"msc":["11Y16","11G20","94A60","14Q05","14K02"],"prefix":"Zbl","series_id":0,"year":"2010"}}],"source":{"book":[],"pages":"385-403","series":[{"acronym":null,"issn":[{"number":"1461-1570","type":"electronic"}],"issue":null,"issue_id":336222,"parallel_title":null,"part":null,"publisher":"Cambridge University Press, Cambridge; London Mathematical Society, London","series_id":2561,"short_title":"LMS J. Comput. Math.","title":"LMS Journal of Computation and Mathematics","volume":"17A","year":"2014"}],"source":"LMS J. Comput. Math. 17A, Spec. Iss., 385-403 (2014)."},"states":[["r","item has references"],["c","is cited"]],"title":{"addition":null,"original":null,"subtitle":null,"title":"Subexponential class group and unit group computation in large degree number fields"},"year":"2014","zbmath_url":"https://zbmath.org/6340507"}]}`
// 	res, _ := ParseToStruct(jsonData)
// 	fmt.Println(res.Result)
// 	if len(res.Result) != 1 {
// 		t.Errorf("Got: %v", len(res.Result))
// 	}
//
// 	//if ent.uni != "Technische Universität Berlin" {
// 	//	t.Errorf("Got: %v", ent.uni)
// 	//}
//
// 	//if ent.year != "1997" {
// 	//	t.Errorf("Got: %v", ent.year)
// 	//}
//
// 	//if ent.title != "Über relative Normgleichungen in algebraischen Zahlkörpern" {
// 	//	t.Errorf("Got: %v", ent.title)
// 	//}
//
// 	//// Test the BibEntry
// 	//expected := bibtex.NewBibTex()
// 	//entry := bibtex.NewBibEntry("thesis", "ClausFieker1997")
// 	//entry.AddField("author", bibtex.NewBibConst("Claus Fieker"))
// 	//entry.AddField("title", bibtex.NewBibConst("{Ü}ber relative {N}ormgleichungen in algebraischen {Z}ahlkörpern"))
// 	//entry.AddField("year", bibtex.NewBibConst("1997"))
// 	//entry.AddField("school", bibtex.NewBibConst("Technische Universität Berlin"))
// 	//expected.AddEntry(entry)
//
// 	//AssertEntriesEqual(t, entry, ent.bib)
// }
