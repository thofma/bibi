package zb

import (
	_ "encoding/json"
	"fmt"
	"github.com/nickng/bibtex"
	_ "github.com/thofma/bibi/util"
	_ "html"
	_ "io"
	_ "net/http"
	_ "net/url"
	_ "os"
	_ "reflect"
	_ "strconv"
	"strings"
	_ "unicode"
)

func ItemToBibEntry(item Item) *bibtex.BibEntry {
	fmt.Println("yoo")
	fmt.Println(item.DocumentType.Code)
	if item.DocumentType.Code == "j" {
		fmt.Println("got a journal article")
		it := ItemToArticle(item)
		fmt.Println(it.PrettyString())
		return it
	} else if item.DocumentType.Code == "a" {
		fmt.Println("got an  inproceedings journal article")
		it := ItemToProceedingsArticle(item)
		fmt.Println(it.PrettyString())
		return it
	} else {
		fmt.Println("not implemented for ", item.DocumentType.Code)
		panic(1)
	}
	entry := bibtex.NewBibEntry("thesis", fmt.Sprintf("%v%v", "ass", "bss"))
	//entry.AddField("author", bibtex.NewBibConst(author))
	//entry.AddField("title", bibtex.NewBibConst(BibtexEncodeTitle(title)))
	//entry.AddField("year", bibtex.NewBibConst(year))
	//entry.AddField("school", bibtex.NewBibConst(university))
	return entry
}

func ItemToArticle(item Item) *bibtex.BibEntry {
	id := ItemGetID(item)
	// first retrieve zbl number to create the label
	label := fmt.Sprintf("zbMATH%v", id)
	entry := bibtex.NewBibEntry("article", label)
	// author
	entry.AddField("author", bibtex.NewBibConst(ItemGetAuthors(item)))
	// title
	// maybe phd.BibtexEncodeTitle?
	entry.AddField("title", bibtex.NewBibConst(ItemGetTitle(item)))
	entry.AddField("journal", bibtex.NewBibConst(ItemGetSeriesTitle(item)))
	issn := ItemGetSeriesISSN(item)
	if issn != "" {
		entry.AddField("issn", bibtex.NewBibConst(issn))
	}
	volume := ItemGetSeriesVolume(item)
	if volume != "" {
		entry.AddField("volume", bibtex.NewBibConst(volume))
	}

	year := ItemGetSeriesYear(item)
	if year != "" {
		entry.AddField("year", bibtex.NewBibConst(year))
	}

	doi, _ := ItemGetDOI(item)
	// don't use the url
	if doi != "" {
		entry.AddField("doi", bibtex.NewBibConst(doi))
	}

	pages := ItemGetSourcePages(item)
	entry.AddField("pages", bibtex.NewBibConst(pages))

	entry.AddField("zbmath", bibtex.NewBibConst(fmt.Sprintf("%v", id)))

	PrintBibtex(entry)

	return entry
}

func ItemGetID(item Item) int {
	return item.ID
}

func ItemGetTitle(item Item) string {
	return item.Title.Title
}

func ItemGetAuthors(item Item) string {
	n := len(item.Contributors.Authors)
	names := make([]string, n)
	for i := 0; i < n; i++ {
		names[i] = item.Contributors.Authors[i].Name
	}
	return strings.Join(names, " and ")
}

func ItemGetSeriesTitle(item Item) string {
	return item.Source.Series[0].ShortTitle
}

func ItemGetSourcePages(item Item) string {
	return item.Source.Pages
}

func ItemGetSeriesISSN(item Item) string {
	if len(item.Source.Series[0].ISSN) > 0 {
		return item.Source.Series[0].ISSN[0].Number
	} else {
		return ""
	}
}

func ItemGetSeriesVolume(item Item) string {
	return item.Source.Series[0].Volume
}

func ItemGetSeriesYear(item Item) string {
	return item.Source.Series[0].Year
}

func ItemGetDOI(item Item) (string, string) {
	for i := 0; i < len(item.Links); i++ {
		l := item.Links[i]
		if l.Type == "doi" {
			return l.Identifier, l.URL
		}
	}
	return "", ""
}

// @inproceedings {MR3952019,
//
//	   AUTHOR = {Hofmann, Tommy and Zhang, Yinan},
//	    TITLE = {Cyclic extensions of prime degree and their {$p$}-adic
//	             regulators},
//	BOOKTITLE = {Proceedings of the {T}hirteenth {A}lgorithmic {N}umber
//	             {T}heory {S}ymposium},
//	   SERIES = {Open Book Ser.},
//	   VOLUME = {2},
//	    PAGES = {311--323},
//	PUBLISHER = {Math. Sci. Publ., Berkeley, CA},
//	     YEAR = {2019},
//	     ISBN = {978-1-935107-03-3; 978-1-935107-02-6},
//	  MRCLASS = {11Y40 (11K41 11R20 11R27)},
//	 MRNUMBER = {3952019},
//
// MRREVIEWER = {Renate\ Scheidler},
// }
func ItemToProceedingsArticle(item Item) *bibtex.BibEntry {
	id := ItemGetID(item)
	// first retrieve zbl number to create the label
	label := fmt.Sprintf("zbMATH%v", id)
	entry := bibtex.NewBibEntry("inproceedings", label)
	// author
	entry.AddField("author", bibtex.NewBibConst(ItemGetAuthors(item)))
	// title
	// maybe phd.BibtexEncodeTitle?
	entry.AddField("title", bibtex.NewBibConst(ItemGetTitle(item)))
	entry.AddField("journal", bibtex.NewBibConst(ItemGetSeriesTitle(item)))
	issn := ItemGetSeriesISSN(item)
	if issn != "" {
		entry.AddField("issn", bibtex.NewBibConst(issn))
	}
	volume := ItemGetSeriesVolume(item)
	if volume != "" {
		entry.AddField("volume", bibtex.NewBibConst(volume))
	}

	year := ItemGetSeriesYear(item)
	if year != "" {
		entry.AddField("year", bibtex.NewBibConst(year))
	}

	doi, _ := ItemGetDOI(item)
	// don't use the url
	if doi != "" {
		entry.AddField("doi", bibtex.NewBibConst(doi))
	}

	pages := ItemGetSourcePages(item)
	entry.AddField("pages", bibtex.NewBibConst(pages))

	entry.AddField("zbmath", bibtex.NewBibConst(fmt.Sprintf("%v", id)))

	PrintBibtex(entry)

	return entry
}

func PrintBibtex(bib *bibtex.BibEntry) {
	if bib.Type == "article" {
		list := []string{"author", "title", "journal", "volme", "year", "pages", "issn", "doi", "zbmath"}
		//var a string
		fmt.Print("@article{", strings.TrimSpace(bib.CiteName), "}\n")
		for a := range list {
			if bib.Fields[list[a]] != nil {
				fmt.Print("  ", list[a], " = ", "{", bib.Fields[list[a]], "},\n")
			}
		}
		fmt.Print("}\n")
	}
	return
}

// Example article:
// @article{zbMATH06340507,
// author = {Biasse, Jean-Fran{\c{c}}ois and Fieker, Claus},
// title = {Subexponential class group and unit group computation in large degree number fields},
// fjournal = {LMS Journal of Computation and Mathematics},
// journal = {LMS J. Comput. Math.},
// issn = {1461-1570},
// volume = {17A},
// pages = {385--403},
// year = {2014},
// language = {English},
// doi = {10.1112/S1461157014000345},
// keywords = {11Y40,11R29,11R27},
// zbMATH = {6340507},
// Zbl = {1369.11103}
//}
//
// @article {MR3531231,
//     AUTHOR = {Hofmann, Tommy and Zhang, Yinan},
//      TITLE = {Valuations of {$p$}-adic regulators of cyclic cubic fields},
//    JOURNAL = {J. Number Theory},
//   FJOURNAL = {Journal of Number Theory},
//     VOLUME = {169},
//       YEAR = {2016},
//      PAGES = {86--102},
//       ISSN = {0022-314X},
//    MRCLASS = {11Y40 (11K41 11R16 11R27)},
//   MRNUMBER = {3531231},
// MRREVIEWER = {Ken Yamamura},
//        DOI = {10.1016/j.jnt.2016.05.016},
//        URL = {https://doi.org/10.1016/j.jnt.2016.05.016},
// }
