package zb

import (
	"strings"
	"testing"

	"github.com/nickng/bibtex"
)

var ExampleResponse = `
  { "result": [
    {
      "biographic_references": [],
      "contributors": {
        "authors": [
          {
            "aliases": [],
            "checked": "1",
            "codes": [
              "hofmann.tommy"
            ],
            "name": "Hofmann, Tommy"
          },
          {
            "aliases": [],
            "checked": "0",
            "codes": [
              "zhang.yinan"
            ],
            "name": "Zhang, Yinan"
          }
        ],
        "author_references": [],
        "editors": []
      },
      "database": "Zbl",
      "datestamp": "2016-08-18T08:45:17Z",
      "document_type": {
        "code": "j",
        "description": "journal article"
      },
      "editorial_contributions": [
        {
          "language": "English",
          "reviewer": {
            "author_code": null,
            "reviewer_id": null,
            "name": null,
            "sign": null
          },
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "contribution_type": "summary"
        }
      ],
      "id": 6617923,
      "identifier": "1409.11141",
      "keywords": [
        "\\(p\\)-adic regulator",
        "distribution of \\(p\\)-adic regulators",
        "cubic fields"
      ],
      "language": {
        "languages": [
          "English"
        ],
        "addition": [null]
      },
      "license": [],
      "links": [
        {
          "identifier": "10.1016/j.jnt.2016.05.016",
          "type": "doi",
          "url": "https://doi.org/10.1016/j.jnt.2016.05.016"
        },
        {
          "identifier": "1701.00340",
          "type": "arxiv",
          "url": "https://arxiv.org/abs/1701.00340"
        }
      ],
      "msc": [
        {
          "code": "11Y40",
          "scheme": "msc2020",
          "text": "Algebraic number theory computations"
        },
        {
          "code": "11R16",
          "scheme": "msc2020",
          "text": "Cubic and quartic extensions"
        },
        {
          "code": "11R27",
          "scheme": "msc2020",
          "text": "Units and factorization"
        }
      ],
      "references": [
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "belabas.karim",
              "friedman.eduardo-c",
              "diaz-y-diaz.francisco"
            ],
            "document_id": 5240465,
            "msc": [
              "11R29",
              "11R47",
              "11M36",
              "11Y40"
            ],
            "prefix": "Zbl",
            "series_id": 240,
            "year": "2008"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "biasse.jean-francois",
              "fieker.claus"
            ],
            "document_id": 6488035,
            "msc": [
              "11Y40",
              "11R29",
              "11R27"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "2013"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "buchmann.johannes-a"
            ],
            "document_id": 4200333,
            "msc": [
              "11Y40",
              "11R29",
              "11Y16",
              "68Q25"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "1990"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "coates.john-h"
            ],
            "document_id": 3609817,
            "msc": [
              "11S40",
              "11R80",
              "11R18",
              "11R23"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "1977"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "cohen.henri"
            ],
            "document_id": 435565,
            "msc": [
              "11Y40",
              "11-02",
              "11Y16",
              "11Y05",
              "11Y11",
              "11Rxx"
            ],
            "prefix": "Zbl",
            "series_id": 3917,
            "year": "1993"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "cohen.henri"
            ],
            "document_id": 1391020,
            "msc": [
              "11Y40",
              "11-02",
              "11Y16",
              "11R37"
            ],
            "prefix": "Zbl",
            "series_id": 3917,
            "year": "2000"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "diaz-y-diaz.francisco",
              "olivier.michel",
              "cohen.henri"
            ],
            "document_id": 1077123,
            "msc": [
              "68W30",
              "11Y40",
              "68W10",
              "11R27",
              "11R29",
              "11Y16"
            ],
            "prefix": "Zbl",
            "series_id": 1072,
            "year": "1997"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "lenstra.hendrik-w-jun",
              "cohen.henri"
            ],
            "document_id": 3889661,
            "msc": [
              "11R29",
              "11R11",
              "11Y40",
              "11R16",
              "11R18"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "1984"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "martinet.jacques",
              "cohen.henri"
            ],
            "document_id": 4019177,
            "msc": [
              "11R29",
              "11-04",
              "11Y40",
              "11R11",
              "11R18"
            ],
            "prefix": "Zbl",
            "series_id": 240,
            "year": "1987"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "fieker.claus"
            ],
            "document_id": 1597968,
            "msc": [
              "11Y40",
              "11R37"
            ],
            "prefix": "Zbl",
            "series_id": 240,
            "year": "2001"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "fieker.claus",
              "zhang.yinan"
            ],
            "document_id": 6601144,
            "msc": [
              "11R29",
              "11S40",
              "11Y35",
              "11Y70"
            ],
            "prefix": "Zbl",
            "series_id": 2561,
            "year": "2016"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "hakkarainen.tuomas"
            ],
            "document_id": 5813059,
            "msc": [
              "11Y40",
              "11R20",
              "11R29",
              "11R27"
            ],
            "prefix": "Zbl",
            "series_id": 240,
            "year": "2009"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "iwasawa.kenkichi"
            ],
            "document_id": 3373863,
            "msc": [
              "11S40",
              "11-02",
              "11M38"
            ],
            "prefix": "Zbl",
            "series_id": 3738,
            "year": "1972"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "malle.gunter"
            ],
            "document_id": 2119347,
            "msc": [
              "11R32",
              "11R47",
              "12F10"
            ],
            "prefix": "Zbl",
            "series_id": 1825,
            "year": "2004"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "malle.gunter"
            ],
            "document_id": 5880814,
            "msc": [
              "11R29",
              "11R16",
              "11Y40"
            ],
            "prefix": "Zbl",
            "series_id": 512,
            "year": "2008"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "malle.gunter"
            ],
            "document_id": 6074863,
            "msc": [
              "11R29",
              "11R16",
              "11R21",
              "11R58",
              "11Y40"
            ],
            "prefix": "Zbl",
            "series_id": 1825,
            "year": "2010"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "marko.frantisek"
            ],
            "document_id": 977858,
            "msc": [
              "11R27",
              "11R20"
            ],
            "prefix": "Zbl",
            "series_id": 272,
            "year": "1996"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [
              "miki.hiroo"
            ],
            "document_id": 4006375,
            "msc": [
              "11R18",
              "11R27",
              "11R34"
            ],
            "prefix": "Zbl",
            "series_id": 512,
            "year": "1987"
          }
        },
        {
          "doi": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "position": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "zbmath": {
            "author_codes": [],
            "document_id": null,
            "msc": [],
            "prefix": null,
            "series_id": 0,
            "year": null
          }
        }
      ],
      "source": {
        "book": [],
        "pages": "86-102",
        "series": [
          {
            "acronym": null,
            "issn": [
              {
                "number": "0022-314X",
                "type": "print"
              },
              {
                "number": "1096-1658",
                "type": "electronic"
              }
            ],
            "issue": null,
            "issue_id": 358319,
            "parallel_title": null,
            "part": null,
            "publisher": "Elsevier (Academic Press), San Diego, CA",
            "series_id": 512,
            "short_title": "J. Number Theory",
            "title": "Journal of Number Theory",
            "volume": "169",
            "year": "2016"
          }
        ],
        "source": "J. Number Theory 169, 86-102 (2016)."
      },
      "states": [
        [
          "r",
          "item has references"
        ],
        [
          "c",
          "is cited"
        ]
      ],
      "title": {
        "addition": null,
        "original": null,
        "subtitle": null,
        "title": "Valuations of \\(p\\)-adic regulators of cyclic cubic fields"
      },
      "year": "2016",
      "zbmath_url": "https://zbmath.org/6617923"
    },
    {
      "biographic_references": [],
      "contributors": {
        "authors": [
          {
            "aliases": [],
            "checked": "1",
            "codes": [
              "hofmann.tommy"
            ],
            "name": "Hofmann, Tommy"
          },
          {
            "aliases": [],
            "checked": "0",
            "codes": [
              "zhang.yinan"
            ],
            "name": "Zhang, Yinan"
          }
        ],
        "author_references": [],
        "editors": []
      },
      "database": "Zbl",
      "datestamp": "2023-08-02T14:27:11Z",
      "document_type": {
        "code": "a",
        "description": "serial article"
      },
      "editorial_contributions": [
        {
          "language": "English",
          "reviewer": {
            "author_code": null,
            "reviewer_id": null,
            "name": null,
            "sign": null
          },
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "contribution_type": "summary"
        }
      ],
      "id": 7721132,
      "identifier": "1531.11123",
      "keywords": [
        "\\(p\\)-adic regulator",
        "distribution of \\(p\\)-adic regulators"
      ],
      "language": {
        "languages": [
          "English"
        ],
        "addition": [null]
      },
      "license": [],
      "links": [
        {
          "identifier": "10.2140/obs.2019.2.311",
          "type": "doi",
          "url": "https://doi.org/10.2140/obs.2019.2.311"
        }
      ],
      "msc": [
        {
          "code": "11Y40",
          "scheme": "msc2020",
          "text": "Algebraic number theory computations"
        },
        {
          "code": "11K41",
          "scheme": "msc2020",
          "text": "Continuous, \\(p\\)-adic and abstract analogues"
        },
        {
          "code": "11R20",
          "scheme": "msc2020",
          "text": "Other abelian and metabelian extensions"
        },
        {
          "code": "11R27",
          "scheme": "msc2020",
          "text": "Units and factorization"
        }
      ],
      "references": [
        {
          "doi": "10.1006/jsco.1996.0125",
          "position": "1",
          "text": "Bosma, Wieb; Cannon, John; Playoust, Catherine, The Magma algebra system, I : The user language, J. Symbolic Comput., 24, 3-4, 235, 1997; Bosma, Wieb; Cannon, John; Playoust, Catherine, The Magma algebra system, I : The user language, J. Symbolic Comput., 24, 3-4, 235, 1997",
          "zbmath": {
            "author_codes": [
              "bosma.wieb",
              "playoust.catherine",
              "cannon.john-j"
            ],
            "document_id": 1077111,
            "msc": [
              "68W30"
            ],
            "prefix": "Zbl",
            "series_id": 1072,
            "year": "1997"
          }
        },
        {
          "doi": null,
          "position": "2",
          "text": "Brumer, Armand, On the units of algebraic number fields, Mathematika, 14, 121, 1967; Brumer, Armand, On the units of algebraic number fields, Mathematika, 14, 121, 1967",
          "zbmath": {
            "author_codes": [
              "brumer.armand"
            ],
            "document_id": 3272326,
            "msc": [],
            "prefix": "Zbl",
            "series_id": 580,
            "year": "1967"
          }
        },
        {
          "doi": null,
          "position": "3",
          "text": "Buchmann, Johannes, A subexponential algorithm for the determination of class groups and regulators of algebraic number fields, Séminaire de Théorie des Nombres. Progr. Math., 91, 27, 1990; Buchmann, Johannes, A subexponential algorithm for the determination of class groups and regulators of algebraic number fields, Séminaire de Théorie des Nombres. Progr. Math., 91, 27, 1990",
          "zbmath": {
            "author_codes": [
              "buchmann.johannes-a"
            ],
            "document_id": 4200333,
            "msc": [
              "11Y40",
              "11R29",
              "11Y16",
              "68Q25"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "1990"
          }
        },
        {
          "doi": null,
          "position": "4",
          "text": "Coates, John, p-adic L-functions and Iwasawa’s theory, Algebraic number fields : L-functions and Galois properties (Proc. Sympos., Univ. Durham), 269, 1977; Coates, John, p-adic L-functions and Iwasawa’s theory, Algebraic number fields : L-functions and Galois properties (Proc. Sympos., Univ. Durham), 269, 1977",
          "zbmath": {
            "author_codes": [
              "coates.john-h"
            ],
            "document_id": 3609817,
            "msc": [
              "11S40",
              "11R80",
              "11R18",
              "11R23"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "1977"
          }
        },
        {
          "doi": null,
          "position": "5",
          "text": "Davis, Philip J., Circulant matrices, 1979; Davis, Philip J., Circulant matrices, 1979",
          "zbmath": {
            "author_codes": [
              "davis.philip-j"
            ],
            "document_id": 3650737,
            "msc": [
              "15B57",
              "15-02",
              "65F30"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "1979"
          }
        },
        {
          "doi": null,
          "position": "6",
          "text": "Fieker, Claus, Computing class fields via the Artin map, Math. Comp., 70, 235, 1293, 2001; Fieker, Claus, Computing class fields via the Artin map, Math. Comp., 70, 235, 1293, 2001",
          "zbmath": {
            "author_codes": [
              "fieker.claus"
            ],
            "document_id": 1597968,
            "msc": [
              "11Y40",
              "11R37"
            ],
            "prefix": "Zbl",
            "series_id": 240,
            "year": "2001"
          }
        },
        {
          "doi": null,
          "position": "7",
          "text": "Fieker, Claus; Zhang, Yinan, An application of the p-adic analytic class number formula, LMS J. Comput. Math., 19, 1, 217, 2016; Fieker, Claus; Zhang, Yinan, An application of the p-adic analytic class number formula, LMS J. Comput. Math., 19, 1, 217, 2016",
          "zbmath": {
            "author_codes": [
              "fieker.claus",
              "zhang.yinan"
            ],
            "document_id": 6601144,
            "msc": [
              "11R29",
              "11S40",
              "11Y35",
              "11Y70"
            ],
            "prefix": "Zbl",
            "series_id": 2561,
            "year": "2016"
          }
        },
        {
          "doi": null,
          "position": "8",
          "text": "Hofmann, Tommy; Zhang, Yinan, Valuations of p-adic regulators of cyclic cubic fields, J. Number Theory, 169, 86, 2016; Hofmann, Tommy; Zhang, Yinan, Valuations of p-adic regulators of cyclic cubic fields, J. Number Theory, 169, 86, 2016",
          "zbmath": {
            "author_codes": [
              "hofmann.tommy",
              "zhang.yinan"
            ],
            "document_id": 6617923,
            "msc": [
              "11Y40",
              "11R16",
              "11R27"
            ],
            "prefix": "Zbl",
            "series_id": 512,
            "year": "2016"
          }
        },
        {
          "doi": null,
          "position": "9",
          "text": "Iwasawa, Kenkichi, Lectures on p-adic L-functions. Annals of Mathematics Studies, 74, 1972; Iwasawa, Kenkichi, Lectures on p-adic L-functions. Annals of Mathematics Studies, 74, 1972",
          "zbmath": {
            "author_codes": [
              "iwasawa.kenkichi"
            ],
            "document_id": 3373863,
            "msc": [
              "11S40",
              "11-02",
              "11M38"
            ],
            "prefix": "Zbl",
            "series_id": 3738,
            "year": "1972"
          }
        },
        {
          "doi": null,
          "position": "10",
          "text": "Lang, Serge, Algebraic number theory. Graduate Texts in Mathematics, 110, 1994; Lang, Serge, Algebraic number theory. Graduate Texts in Mathematics, 110, 1994",
          "zbmath": {
            "author_codes": [
              "lang.serge"
            ],
            "document_id": 611919,
            "msc": [
              "11-02",
              "11-01",
              "11Rxx",
              "11Sxx"
            ],
            "prefix": "Zbl",
            "series_id": 3917,
            "year": "1994"
          }
        },
        {
          "doi": null,
          "position": "11",
          "text": "Leopoldt, Heinrich-Wolfgang, Zur Arithmetik in abelschen Zahlkörpern, J. Reine Angew. Math., 209, 54, 1962; Leopoldt, Heinrich-Wolfgang, Zur Arithmetik in abelschen Zahlkörpern, J. Reine Angew. Math., 209, 54, 1962",
          "zbmath": {
            "author_codes": [
              "leopoldt.heinrich-wolfgang"
            ],
            "document_id": 3323990,
            "msc": [
              "11R20",
              "11R29",
              "11-02"
            ],
            "prefix": "Zbl",
            "series_id": 520,
            "year": "1962"
          }
        },
        {
          "doi": null,
          "position": "12",
          "text": "Marko, František, On the existence of p-units and Minkowski units in totally real cyclic fields, Abh. Math. Sem. Univ. Hamburg, 66, 89, 1996; Marko, František, On the existence of p-units and Minkowski units in totally real cyclic fields, Abh. Math. Sem. Univ. Hamburg, 66, 89, 1996",
          "zbmath": {
            "author_codes": [
              "marko.frantisek"
            ],
            "document_id": 977858,
            "msc": [
              "11R27",
              "11R20"
            ],
            "prefix": "Zbl",
            "series_id": 272,
            "year": "1996"
          }
        },
        {
          "doi": null,
          "position": "13",
          "text": "Miki, Hiroo, On the Leopoldt conjecture on the p-adic regulators, J. Number Theory, 26, 2, 117, 1987; Miki, Hiroo, On the Leopoldt conjecture on the p-adic regulators, J. Number Theory, 26, 2, 117, 1987",
          "zbmath": {
            "author_codes": [
              "miki.hiroo"
            ],
            "document_id": 4006375,
            "msc": [
              "11R18",
              "11R27",
              "11R34"
            ],
            "prefix": "Zbl",
            "series_id": 512,
            "year": "1987"
          }
        },
        {
          "doi": null,
          "position": "14",
          "text": "Schirokauer, Oliver, Discrete logarithms and local units, Philos. Trans. Roy. Soc. London Ser. A, 345, 1676, 409, 1993; Schirokauer, Oliver, Discrete logarithms and local units, Philos. Trans. Roy. Soc. London Ser. A, 345, 1676, 409, 1993",
          "zbmath": {
            "author_codes": [
              "schirokauer.oliver"
            ],
            "document_id": 549320,
            "msc": [
              "11Y05",
              "11R27",
              "11Y40",
              "11Y16"
            ],
            "prefix": "Zbl",
            "series_id": 1770,
            "year": "1993"
          }
        },
        {
          "doi": null,
          "position": "15",
          "text": "Serre, Jean-Pierre, Local fields. Graduate Texts in Mathematics, 67, 1979; Serre, Jean-Pierre, Local fields. Graduate Texts in Mathematics, 67, 1979",
          "zbmath": {
            "author_codes": [
              "serre.jean-pierre"
            ],
            "document_id": 3657912,
            "msc": [
              "11Sxx",
              "11-01",
              "11-02",
              "14Gxx"
            ],
            "prefix": "Zbl",
            "series_id": 3917,
            "year": "1979"
          }
        }
      ],
      "source": {
        "book": [
          {
            "book_id": 7083365,
            "isbn": [
              {
                "number": "978-1-935107-02-6",
                "type": "print"
              },
              {
                "number": "978-1-935107-03-3",
                "type": "ebook"
              }
            ],
            "publisher": "Berkeley, CA: Mathematical Sciences Publishers (MSP)",
            "year": "2019"
          }
        ],
        "pages": "311-323",
        "series": [],
        "source": "Scheidler, Renate (ed.) et al., ANTS XIII. Proceedings of the thirteenth algorithmic number theory symposium, University of Wisconsin-Madison, WI, USA, July 16--20, 2018. Berkeley, CA: Mathematical Sciences Publishers (MSP). Open Book Ser. 2, 311-323 (2019)."
      },
      "states": [
        [
          "r",
          "item has references"
        ]
      ],
      "title": {
        "addition": null,
        "original": null,
        "subtitle": null,
        "title": "Cyclic extensions of prime degree and their \\(p\\)-adic regulators"
      },
      "year": "2019",
      "zbmath_url": "https://zbmath.org/7721132"
    }
  ],
  "status": {
    "execution": "successful request",
    "execution_bool": true,
    "internal_code": "ok",
    "last_id": null,
    "nr_total_results": 2,
    "nr_request_results": 2,
    "query_execution_time_in_seconds": 0.155877113342285,
    "status_code": 200,
    "time_stamp": "2025-09-01T06:40:16Z"
  }
}
`

func AssertEntriesEqual(t *testing.T, a, b *bibtex.BibEntry) {
	t.Helper()
	if a.Type != b.Type {
		t.Errorf("type = %q, want %q", b.Type, a.Type)
	}
	if a.CiteName != b.CiteName {
		t.Errorf("cite name = %q, want %q", b.CiteName, a.CiteName)
	}
	if len(a.Fields) != len(b.Fields) {
		t.Fatalf("field count = %d, want %d", len(b.Fields), len(a.Fields))
	}
	for key, want := range a.Fields {
		got, ok := b.Fields[key]
		if !ok {
			t.Fatalf("missing field %q", key)
		}
		if want.String() != got.String() {
			t.Fatalf("field %q = %q, want %q", key, got.String(), want.String())
		}
	}
}

func AssertValidBibTeX(t *testing.T, entry *bibtex.BibEntry) {
	t.Helper()
	parsed, err := bibtex.Parse(strings.NewReader(entry.PrettyString()))
	if err != nil {
		t.Fatalf("generated BibTeX does not parse: %v\n%s", err, entry.PrettyString())
	}
	if len(parsed.Entries) != 1 {
		t.Fatalf("generated BibTeX has %d entries, want 1", len(parsed.Entries))
	}
}

func TestJournalArticle(t *testing.T) {
	resp, err := ParseToStruct(ExampleResponse)
	if err != nil {
		t.Fatalf("ParseToStruct() error = %v", err)
	}
	bib, err := ItemToBibEntry(resp.Result[0], resp.Result...)
	if err != nil {
		t.Fatalf("ItemToBibEntry() error = %v", err)
	}

	entry := bibtex.NewBibEntry("article", "zbMATH6617923")
	entry.AddField("author", bibtex.NewBibConst("Hofmann, Tommy and Zhang, Yinan"))
	entry.AddField("title", bibtex.NewBibConst("{Valuations of \\(p\\)-adic regulators of cyclic cubic fields}"))
	entry.AddField("year", bibtex.NewBibConst("2016"))
	entry.AddField("journal", bibtex.NewBibConst("J. Number Theory"))
	entry.AddField("fjournal", bibtex.NewBibConst("Journal of Number Theory"))
	entry.AddField("volume", bibtex.NewBibConst("169"))
	entry.AddField("pages", bibtex.NewBibConst("86--102"))
	entry.AddField("issn", bibtex.NewBibConst("0022-314X"))
	entry.AddField("doi", bibtex.NewBibConst("10.1016/j.jnt.2016.05.016"))
	entry.AddField("zbmath", bibtex.NewBibConst("6617923"))
	AssertEntriesEqual(t, entry, bib)
	AssertValidBibTeX(t, bib)
}

var ExampleProceedings = `
{
  "result": [
    {
      "biographic_references": [],
      "contributors": {
        "authors": [
          {
            "aliases": [],
            "checked": "1",
            "codes": [
              "hofmann.tommy"
            ],
            "name": "Hofmann, Tommy"
          },
          {
            "aliases": [],
            "checked": "0",
            "codes": [
              "zhang.yinan"
            ],
            "name": "Zhang, Yinan"
          }
        ],
        "author_references": [],
        "editors": []
      },
      "database": "Zbl",
      "datestamp": "2023-08-02T14:27:11Z",
      "document_type": {
        "code": "a",
        "description": "serial article"
      },
      "editorial_contributions": [
        {
          "language": "English",
          "reviewer": {
            "author_code": null,
            "reviewer_id": null,
            "name": null,
            "sign": null
          },
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "contribution_type": "summary"
        }
      ],
      "id": 7721132,
      "identifier": "1531.11123",
      "keywords": [
        "\\(p\\)-adic regulator",
        "distribution of \\(p\\)-adic regulators"
      ],
      "language": {
        "languages": [
          "English"
        ],
        "addition": [null]
      },
      "license": [],
      "links": [
        {
          "identifier": "10.2140/obs.2019.2.311",
          "type": "doi",
          "url": "https://doi.org/10.2140/obs.2019.2.311"
        }
      ],
      "msc": [
        {
          "code": "11Y40",
          "scheme": "msc2020",
          "text": "Algebraic number theory computations"
        },
        {
          "code": "11K41",
          "scheme": "msc2020",
          "text": "Continuous, \\(p\\)-adic and abstract analogues"
        },
        {
          "code": "11R20",
          "scheme": "msc2020",
          "text": "Other abelian and metabelian extensions"
        },
        {
          "code": "11R27",
          "scheme": "msc2020",
          "text": "Units and factorization"
        }
      ],
      "references": [
        {
          "doi": "10.1006/jsco.1996.0125",
          "position": "1",
          "text": "Bosma, Wieb; Cannon, John; Playoust, Catherine, The Magma algebra system, I : The user language, J. Symbolic Comput., 24, 3-4, 235, 1997; Bosma, Wieb; Cannon, John; Playoust, Catherine, The Magma algebra system, I : The user language, J. Symbolic Comput., 24, 3-4, 235, 1997",
          "zbmath": {
            "author_codes": [
              "bosma.wieb",
              "playoust.catherine",
              "cannon.john-j"
            ],
            "document_id": 1077111,
            "msc": [
              "68W30"
            ],
            "prefix": "Zbl",
            "series_id": 1072,
            "year": "1997"
          }
        },
        {
          "doi": null,
          "position": "2",
          "text": "Brumer, Armand, On the units of algebraic number fields, Mathematika, 14, 121, 1967; Brumer, Armand, On the units of algebraic number fields, Mathematika, 14, 121, 1967",
          "zbmath": {
            "author_codes": [
              "brumer.armand"
            ],
            "document_id": 3272326,
            "msc": [],
            "prefix": "Zbl",
            "series_id": 580,
            "year": "1967"
          }
        },
        {
          "doi": null,
          "position": "3",
          "text": "Buchmann, Johannes, A subexponential algorithm for the determination of class groups and regulators of algebraic number fields, Séminaire de Théorie des Nombres. Progr. Math., 91, 27, 1990; Buchmann, Johannes, A subexponential algorithm for the determination of class groups and regulators of algebraic number fields, Séminaire de Théorie des Nombres. Progr. Math., 91, 27, 1990",
          "zbmath": {
            "author_codes": [
              "buchmann.johannes-a"
            ],
            "document_id": 4200333,
            "msc": [
              "11Y40",
              "11R29",
              "11Y16",
              "68Q25"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "1990"
          }
        },
        {
          "doi": null,
          "position": "4",
          "text": "Coates, John, p-adic L-functions and Iwasawa’s theory, Algebraic number fields : L-functions and Galois properties (Proc. Sympos., Univ. Durham), 269, 1977; Coates, John, p-adic L-functions and Iwasawa’s theory, Algebraic number fields : L-functions and Galois properties (Proc. Sympos., Univ. Durham), 269, 1977",
          "zbmath": {
            "author_codes": [
              "coates.john-h"
            ],
            "document_id": 3609817,
            "msc": [
              "11S40",
              "11R80",
              "11R18",
              "11R23"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "1977"
          }
        },
        {
          "doi": null,
          "position": "5",
          "text": "Davis, Philip J., Circulant matrices, 1979; Davis, Philip J., Circulant matrices, 1979",
          "zbmath": {
            "author_codes": [
              "davis.philip-j"
            ],
            "document_id": 3650737,
            "msc": [
              "15B57",
              "15-02",
              "65F30"
            ],
            "prefix": "Zbl",
            "series_id": 0,
            "year": "1979"
          }
        },
        {
          "doi": null,
          "position": "6",
          "text": "Fieker, Claus, Computing class fields via the Artin map, Math. Comp., 70, 235, 1293, 2001; Fieker, Claus, Computing class fields via the Artin map, Math. Comp., 70, 235, 1293, 2001",
          "zbmath": {
            "author_codes": [
              "fieker.claus"
            ],
            "document_id": 1597968,
            "msc": [
              "11Y40",
              "11R37"
            ],
            "prefix": "Zbl",
            "series_id": 240,
            "year": "2001"
          }
        },
        {
          "doi": null,
          "position": "7",
          "text": "Fieker, Claus; Zhang, Yinan, An application of the p-adic analytic class number formula, LMS J. Comput. Math., 19, 1, 217, 2016; Fieker, Claus; Zhang, Yinan, An application of the p-adic analytic class number formula, LMS J. Comput. Math., 19, 1, 217, 2016",
          "zbmath": {
            "author_codes": [
              "fieker.claus",
              "zhang.yinan"
            ],
            "document_id": 6601144,
            "msc": [
              "11R29",
              "11S40",
              "11Y35",
              "11Y70"
            ],
            "prefix": "Zbl",
            "series_id": 2561,
            "year": "2016"
          }
        },
        {
          "doi": null,
          "position": "8",
          "text": "Hofmann, Tommy; Zhang, Yinan, Valuations of p-adic regulators of cyclic cubic fields, J. Number Theory, 169, 86, 2016; Hofmann, Tommy; Zhang, Yinan, Valuations of p-adic regulators of cyclic cubic fields, J. Number Theory, 169, 86, 2016",
          "zbmath": {
            "author_codes": [
              "hofmann.tommy",
              "zhang.yinan"
            ],
            "document_id": 6617923,
            "msc": [
              "11Y40",
              "11R16",
              "11R27"
            ],
            "prefix": "Zbl",
            "series_id": 512,
            "year": "2016"
          }
        },
        {
          "doi": null,
          "position": "9",
          "text": "Iwasawa, Kenkichi, Lectures on p-adic L-functions. Annals of Mathematics Studies, 74, 1972; Iwasawa, Kenkichi, Lectures on p-adic L-functions. Annals of Mathematics Studies, 74, 1972",
          "zbmath": {
            "author_codes": [
              "iwasawa.kenkichi"
            ],
            "document_id": 3373863,
            "msc": [
              "11S40",
              "11-02",
              "11M38"
            ],
            "prefix": "Zbl",
            "series_id": 3738,
            "year": "1972"
          }
        },
        {
          "doi": null,
          "position": "10",
          "text": "Lang, Serge, Algebraic number theory. Graduate Texts in Mathematics, 110, 1994; Lang, Serge, Algebraic number theory. Graduate Texts in Mathematics, 110, 1994",
          "zbmath": {
            "author_codes": [
              "lang.serge"
            ],
            "document_id": 611919,
            "msc": [
              "11-02",
              "11-01",
              "11Rxx",
              "11Sxx"
            ],
            "prefix": "Zbl",
            "series_id": 3917,
            "year": "1994"
          }
        },
        {
          "doi": null,
          "position": "11",
          "text": "Leopoldt, Heinrich-Wolfgang, Zur Arithmetik in abelschen Zahlkörpern, J. Reine Angew. Math., 209, 54, 1962; Leopoldt, Heinrich-Wolfgang, Zur Arithmetik in abelschen Zahlkörpern, J. Reine Angew. Math., 209, 54, 1962",
          "zbmath": {
            "author_codes": [
              "leopoldt.heinrich-wolfgang"
            ],
            "document_id": 3323990,
            "msc": [
              "11R20",
              "11R29",
              "11-02"
            ],
            "prefix": "Zbl",
            "series_id": 520,
            "year": "1962"
          }
        },
        {
          "doi": null,
          "position": "12",
          "text": "Marko, František, On the existence of p-units and Minkowski units in totally real cyclic fields, Abh. Math. Sem. Univ. Hamburg, 66, 89, 1996; Marko, František, On the existence of p-units and Minkowski units in totally real cyclic fields, Abh. Math. Sem. Univ. Hamburg, 66, 89, 1996",
          "zbmath": {
            "author_codes": [
              "marko.frantisek"
            ],
            "document_id": 977858,
            "msc": [
              "11R27",
              "11R20"
            ],
            "prefix": "Zbl",
            "series_id": 272,
            "year": "1996"
          }
        },
        {
          "doi": null,
          "position": "13",
          "text": "Miki, Hiroo, On the Leopoldt conjecture on the p-adic regulators, J. Number Theory, 26, 2, 117, 1987; Miki, Hiroo, On the Leopoldt conjecture on the p-adic regulators, J. Number Theory, 26, 2, 117, 1987",
          "zbmath": {
            "author_codes": [
              "miki.hiroo"
            ],
            "document_id": 4006375,
            "msc": [
              "11R18",
              "11R27",
              "11R34"
            ],
            "prefix": "Zbl",
            "series_id": 512,
            "year": "1987"
          }
        },
        {
          "doi": null,
          "position": "14",
          "text": "Schirokauer, Oliver, Discrete logarithms and local units, Philos. Trans. Roy. Soc. London Ser. A, 345, 1676, 409, 1993; Schirokauer, Oliver, Discrete logarithms and local units, Philos. Trans. Roy. Soc. London Ser. A, 345, 1676, 409, 1993",
          "zbmath": {
            "author_codes": [
              "schirokauer.oliver"
            ],
            "document_id": 549320,
            "msc": [
              "11Y05",
              "11R27",
              "11Y40",
              "11Y16"
            ],
            "prefix": "Zbl",
            "series_id": 1770,
            "year": "1993"
          }
        },
        {
          "doi": null,
          "position": "15",
          "text": "Serre, Jean-Pierre, Local fields. Graduate Texts in Mathematics, 67, 1979; Serre, Jean-Pierre, Local fields. Graduate Texts in Mathematics, 67, 1979",
          "zbmath": {
            "author_codes": [
              "serre.jean-pierre"
            ],
            "document_id": 3657912,
            "msc": [
              "11Sxx",
              "11-01",
              "11-02",
              "14Gxx"
            ],
            "prefix": "Zbl",
            "series_id": 3917,
            "year": "1979"
          }
        }
      ],
      "source": {
        "book": [
          {
            "book_id": 7083365,
            "isbn": [
              {
                "number": "978-1-935107-02-6",
                "type": "print"
              },
              {
                "number": "978-1-935107-03-3",
                "type": "ebook"
              }
            ],
            "publisher": "Berkeley, CA: Mathematical Sciences Publishers (MSP)",
            "year": "2019"
          }
        ],
        "pages": "311-323",
        "series": [],
        "source": "Scheidler, Renate (ed.) et al., ANTS XIII. Proceedings of the thirteenth algorithmic number theory symposium, University of Wisconsin-Madison, WI, USA, July 16--20, 2018. Berkeley, CA: Mathematical Sciences Publishers (MSP). Open Book Ser. 2, 311-323 (2019)."
      },
      "states": [
        [
          "r",
          "item has references"
        ]
      ],
      "title": {
        "addition": null,
        "original": null,
        "subtitle": null,
        "title": "Cyclic extensions of prime degree and their \\(p\\)-adic regulators"
      },
      "year": "2019",
      "zbmath_url": "https://zbmath.org/7721132"
    },
    {
      "biographic_references": [],
      "contributors": {
        "authors": [],
        "author_references": [],
        "editors": [
          {
            "aliases": [],
            "checked": "0",
            "codes": [
              "scheidler.renate"
            ],
            "name": "Scheidler, Renate"
          },
          {
            "aliases": [],
            "checked": "1",
            "codes": [
              "sorenson.jonathan-p"
            ],
            "name": "Sorenson, Jonathan"
          }
        ]
      },
      "database": "Zbl",
      "datestamp": "2019-07-19T11:56:41Z",
      "document_type": {
        "code": "b",
        "description": "book / book article"
      },
      "editorial_contributions": [
        {
          "language": "English",
          "reviewer": {
            "author_code": null,
            "reviewer_id": null,
            "name": null,
            "sign": null
          },
          "text": "zbMATH Open Web Interface contents unavailable due to conflicting licenses.",
          "contribution_type": "editorial"
        }
      ],
      "id": 7083365,
      "identifier": "1416.11009",
      "keywords": [null],
      "language": {
        "languages": [
          "English"
        ],
        "addition": [null]
      },
      "license": [],
      "links": [
        {
          "identifier": "10.2140/obs.2019.2-1",
          "type": "doi",
          "url": "https://doi.org/10.2140/obs.2019.2-1"
        },
        {
          "identifier": "msp.org/obs/2019/2-1/obs-v2-n1-s.pdf",
          "type": "https",
          "url": "https://msp.org/obs/2019/2-1/obs-v2-n1-s.pdf"
        }
      ],
      "msc": [
        {
          "code": "11-06",
          "scheme": "msc2020",
          "text": "Proceedings, conferences, collections, etc. pertaining to number theory"
        },
        {
          "code": "11Yxx",
          "scheme": "msc2020",
          "text": "Computational number theory"
        },
        {
          "code": "00B25",
          "scheme": "msc2020",
          "text": "Proceedings of conferences of miscellaneous specific interest"
        }
      ],
      "references": [],
      "source": {
        "book": [
          {
            "book_id": 7083365,
            "isbn": [
              {
                "number": "978-1-935107-02-6",
                "type": "print"
              },
              {
                "number": "978-1-935107-03-3",
                "type": "ebook"
              }
            ],
            "publisher": "Berkeley, CA: Mathematical Sciences Publishers (MSP)",
            "year": "2019"
          }
        ],
        "pages": "x, 478~p.",
        "series": [
          {
            "acronym": null,
            "issn": [
              {
                "number": "2329-9061",
                "type": "print"
              },
              {
                "number": "2329-907X",
                "type": "electronic"
              }
            ],
            "issue": null,
            "issue_id": 442509,
            "parallel_title": null,
            "part": null,
            "publisher": "Mathematical Sciences Publishers (MSP), Berkeley, CA",
            "series_id": 8505,
            "short_title": "Open Book Ser.",
            "title": "The Open Book Series",
            "volume": "2",
            "year": "2019"
          }
        ],
        "source": "The Open Book Series 2. Berkeley, CA: Mathematical Sciences Publishers (MSP) (ISBN 978-1-935107-02-6/print; 978-1-935107-03-3/ebook). x, 478~p., open access (2019)."
      },
      "states": [
        [
          "c",
          "is cited"
        ]
      ],
      "title": {
        "addition": null,
        "original": null,
        "subtitle": null,
        "title": "ANTS XIII. Proceedings of the thirteenth algorithmic number theory symposium, University of Wisconsin-Madison, WI, USA, July 16--20, 2018"
      },
      "year": "2019",
      "zbmath_url": "https://zbmath.org/7083365"
    }
  ],
  "status": {
    "execution": "successful request",
    "execution_bool": true,
    "internal_code": "ok",
    "last_id": null,
    "nr_total_results": 2,
    "nr_request_results": 2,
    "query_execution_time_in_seconds": 0.156043291091919,
    "status_code": 200,
    "time_stamp": "2025-09-01T06:49:10Z"
  }
}
`

func TestProceedingArticle(t *testing.T) {
	resp, err := ParseToStruct(ExampleProceedings)
	if err != nil {
		t.Fatalf("ParseToStruct() error = %v", err)
	}
	bib, err := ItemToBibEntry(resp.Result[0], resp.Result...)
	if err != nil {
		t.Fatalf("ItemToBibEntry() error = %v", err)
	}

	entry := bibtex.NewBibEntry("inproceedings", "zbMATH7721132")
	entry.AddField("author", bibtex.NewBibConst("Hofmann, Tommy and Zhang, Yinan"))
	entry.AddField("title", bibtex.NewBibConst("{Cyclic extensions of prime degree and their \\(p\\)-adic regulators}"))
	entry.AddField("booktitle", bibtex.NewBibConst("{ANTS XIII. Proceedings of the thirteenth algorithmic number theory symposium, University of Wisconsin-Madison, WI, USA, July 16--20, 2018}"))
	entry.AddField("editor", bibtex.NewBibConst("Scheidler, Renate and Sorenson, Jonathan"))
	entry.AddField("publisher", bibtex.NewBibConst("Berkeley, CA: Mathematical Sciences Publishers (MSP)"))
	entry.AddField("series", bibtex.NewBibConst("Open Book Ser."))
	entry.AddField("volume", bibtex.NewBibConst("2"))
	entry.AddField("pages", bibtex.NewBibConst("311--323"))
	entry.AddField("year", bibtex.NewBibConst("2019"))
	entry.AddField("isbn", bibtex.NewBibConst("978-1-935107-02-6; 978-1-935107-03-3"))
	entry.AddField("issn", bibtex.NewBibConst("2329-9061"))
	entry.AddField("doi", bibtex.NewBibConst("10.2140/obs.2019.2.311"))
	entry.AddField("zbmath", bibtex.NewBibConst("7721132"))
	AssertEntriesEqual(t, entry, bib)
	AssertValidBibTeX(t, bib)
}

func TestBook(t *testing.T) {
	resp, err := ParseToStruct(ExampleProceedings)
	if err != nil {
		t.Fatalf("ParseToStruct() error = %v", err)
	}
	bib, err := ItemToBibEntry(resp.Result[1], resp.Result...)
	if err != nil {
		t.Fatalf("ItemToBibEntry() error = %v", err)
	}

	entry := bibtex.NewBibEntry("book", "zbMATH7083365")
	entry.AddField("editor", bibtex.NewBibConst("Scheidler, Renate and Sorenson, Jonathan"))
	entry.AddField("title", bibtex.NewBibConst("{ANTS XIII. Proceedings of the thirteenth algorithmic number theory symposium, University of Wisconsin-Madison, WI, USA, July 16--20, 2018}"))
	entry.AddField("publisher", bibtex.NewBibConst("Berkeley, CA: Mathematical Sciences Publishers (MSP)"))
	entry.AddField("series", bibtex.NewBibConst("Open Book Ser."))
	entry.AddField("volume", bibtex.NewBibConst("2"))
	entry.AddField("pages", bibtex.NewBibConst("x, 478~p."))
	entry.AddField("year", bibtex.NewBibConst("2019"))
	entry.AddField("isbn", bibtex.NewBibConst("978-1-935107-02-6; 978-1-935107-03-3"))
	entry.AddField("issn", bibtex.NewBibConst("2329-9061"))
	entry.AddField("doi", bibtex.NewBibConst("10.2140/obs.2019.2-1"))
	entry.AddField("zbmath", bibtex.NewBibConst("7083365"))
	AssertEntriesEqual(t, entry, bib)
	AssertValidBibTeX(t, bib)
}

func TestArticleWithoutSeriesDoesNotPanic(t *testing.T) {
	item := Item{
		DocumentType: DocumentType{Code: "j"},
		ID:           1,
		Title:        Title{Title: "An article without source metadata"},
		Year:         "2026",
	}
	bib, err := ItemToBibEntry(item)
	if err != nil {
		t.Fatalf("ItemToBibEntry() error = %v", err)
	}
	if _, ok := bib.Fields["journal"]; ok {
		t.Error("article without series unexpectedly has a journal field")
	}
	AssertValidBibTeX(t, bib)
}

func TestArXivPreprintWithoutDocumentType(t *testing.T) {
	item := Item{
		Contributors: Contributors{Authors: []Author{
			{Name: "Tommy Hofmann"},
			{Name: "John Nicholson"},
		}},
		Database:   "arXiv",
		ID:         902789343,
		Identifier: "arXiv:2507.15999",
		Links: []Link{{
			Identifier: "2507.15999",
			Type:       "arxiv",
			URL:        "https://arxiv.org/abs/2507.15999",
		}},
		Title: Title{Title: "Exotic presentations of quaternion groups and Wall's D2 problem"},
		Year:  "2025",
	}

	bib, err := ItemToBibEntry(item)
	if err != nil {
		t.Fatalf("ItemToBibEntry() error = %v", err)
	}

	entry := bibtex.NewBibEntry("misc", "zbMATH902789343")
	entry.AddField("author", bibtex.NewBibConst("Tommy Hofmann and John Nicholson"))
	entry.AddField("title", bibtex.NewBibConst("{Exotic presentations of quaternion groups and Wall's D2 problem}"))
	entry.AddField("year", bibtex.NewBibConst("2025"))
	entry.AddField("eprint", bibtex.NewBibConst("2507.15999"))
	entry.AddField("archiveprefix", bibtex.NewBibConst("arXiv"))
	entry.AddField("url", bibtex.NewBibConst("https://arxiv.org/abs/2507.15999"))
	entry.AddField("zbmath", bibtex.NewBibConst("902789343"))
	AssertEntriesEqual(t, entry, bib)
	AssertValidBibTeX(t, bib)
}

func TestProceedingsWithoutBookTitleOmitsField(t *testing.T) {
	item := Item{
		DocumentType: DocumentType{Code: "a"},
		ID:           2,
		Title:        Title{Title: "A proceedings article"},
	}
	bib, err := ItemToBibEntry(item)
	if err != nil {
		t.Fatalf("ItemToBibEntry() error = %v", err)
	}
	if _, ok := bib.Fields["booktitle"]; ok {
		t.Errorf("booktitle = %q, want omitted", bib.Fields["booktitle"])
	}
	AssertValidBibTeX(t, bib)
}

func TestUnsupportedDocumentTypeReturnsError(t *testing.T) {
	_, err := ItemToBibEntry(Item{DocumentType: DocumentType{Code: "x"}})
	if err == nil {
		t.Fatal("ItemToBibEntry() error = nil, want unsupported-type error")
	}
}
