package i18n

import (
	"net/http"
	"strings"

	"tourna-desk/internal/models"
)

type Lang string

const (
	LangTR Lang = "tr"
	LangEN Lang = "en"
)

type Dictionary struct {
	SheetStandings string
	SheetFixtures  string

	// Standings headers
	ColRank      string
	ColStage     string
	ColTeam      string
	ColSchool    string
	ColPlayed    string
	ColWon       string
	ColDrawn     string
	ColLost      string
	ColPoints    string
	ColFor       string
	ColAgainst   string
	ColDiff      string
	ColSetsWon   string
	ColSetsLost  string
	ColSetRatio  string

	// Fixture headers
	ColRound    string
	ColMatchNo  string
	ColDate     string
	ColVenue    string
	ColCourt    string
	ColHomeTeam string
	ColScore    string
	ColAwayTeam string
	ColStatus   string

	// Team Template headers
	TemplateSheetName string
	TemplateColSchool string
	TemplateColTag    string

	// Statuses
	StatusScheduled string
	StatusCompleted string
	StatusBye       string

	// Labels
	LabelRound     string
	LabelBye       string
	LabelGenerated string

	// Sports
	SportFootball   string
	SportFutsal     string
	SportBasketball string
	SportVolleyball string
	SportHandball   string
}

var trDict = Dictionary{
	SheetStandings: "Puan Durumu",
	SheetFixtures:  "Fikstür",

	ColRank:     "Sıra",
	ColStage:    "Grup / Aşama",
	ColTeam:     "Takım",
	ColSchool:   "Okul Adı",
	ColPlayed:   "O",
	ColWon:      "G",
	ColDrawn:    "B",
	ColLost:     "M",
	ColPoints:   "P",
	ColFor:      "A",
	ColAgainst:  "Y",
	ColDiff:     "Av",
	ColSetsWon:  "AS",
	ColSetsLost: "YS",
	ColSetRatio: "SO",

	ColRound:    "Hafta / Tur",
	ColMatchNo:  "Maç No",
	ColDate:     "Tarih / Saat",
	ColVenue:    "Salon / Tesis",
	ColCourt:    "Saha / Kort",
	ColHomeTeam: "Ev Sahibi",
	ColScore:    "Skor",
	ColAwayTeam: "Deplasman",
	ColStatus:   "Durum",

	TemplateSheetName: "Takım Listesi Şablonu",
	TemplateColSchool: "Okul Adı",
	TemplateColTag:    "Takım Harfi (Opsiyonel)",

	StatusScheduled: "Planlandı",
	StatusCompleted: "Tamamlandı",
	StatusBye:       "BAY",

	LabelRound:     "Tur",
	LabelBye:       "BAY",
	LabelGenerated: "Oluşturulma Tarihi",

	SportFootball:   "Futbol",
	SportFutsal:     "Futsal",
	SportBasketball: "Basketbol",
	SportVolleyball: "Voleybol",
	SportHandball:   "Hentbol",
}

var enDict = Dictionary{
	SheetStandings: "Standings",
	SheetFixtures:  "Fixtures",

	ColRank:     "Rank",
	ColStage:    "Group / Stage",
	ColTeam:     "Team",
	ColSchool:   "School Name",
	ColPlayed:   "P",
	ColWon:      "W",
	ColDrawn:    "D",
	ColLost:     "L",
	ColPoints:   "Pts",
	ColFor:      "For",
	ColAgainst:  "Agst",
	ColDiff:     "Diff",
	ColSetsWon:  "SW",
	ColSetsLost: "SL",
	ColSetRatio: "SR",

	ColRound:    "Round",
	ColMatchNo:  "Match #",
	ColDate:     "Date / Time",
	ColVenue:    "Venue",
	ColCourt:    "Court",
	ColHomeTeam: "Home Team",
	ColScore:    "Score",
	ColAwayTeam: "Away Team",
	ColStatus:   "Status",

	TemplateSheetName: "Team Import Template",
	TemplateColSchool: "Okul Adı",
	TemplateColTag:    "Takım Harfi (Opsiyonel)",

	StatusScheduled: "Scheduled",
	StatusCompleted: "Completed",
	StatusBye:       "BYE",

	LabelRound:     "Round",
	LabelBye:       "BYE",
	LabelGenerated: "Generated At",

	SportFootball:   "Football",
	SportFutsal:     "Futsal",
	SportBasketball: "Basketball",
	SportVolleyball: "Volleyball",
	SportHandball:   "Handball",
}

func DetectLanguage(r *http.Request) Lang {
	if r == nil {
		return LangTR
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	if q == "en" || strings.HasPrefix(q, "en-") {
		return LangEN
	}
	if q == "tr" || strings.HasPrefix(q, "tr-") {
		return LangTR
	}

	accept := strings.ToLower(r.Header.Get("Accept-Language"))
	if strings.HasPrefix(accept, "en") {
		return LangEN
	}
	return LangTR
}

func Get(lang Lang) Dictionary {
	if lang == LangEN {
		return enDict
	}
	return trDict
}

func (d Dictionary) FormatStatus(status models.MatchStatus, isBye bool) string {
	if isBye || status == models.StatusByeAdvanced {
		return d.StatusBye
	}
	if status == models.StatusCompleted {
		return d.StatusCompleted
	}
	return d.StatusScheduled
}

func (d Dictionary) SportName(slug models.SportSlug) string {
	switch slug {
	case models.SportFutsal:
		return d.SportFutsal
	case models.SportBasketball:
		return d.SportBasketball
	case models.SportVolleyball:
		return d.SportVolleyball
	case models.SportHandball:
		return d.SportHandball
	default:
		return d.SportFootball
	}
}
