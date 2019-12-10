package country

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

var currencyDomain = "https://prepaid.currconv.com/api/v7"
var currencyAPIKey = "pr_5cfdec6210844621a3ed904824b6e54b"
var sherpaDomain = "https://requirements-api.joinsherpa.com/v2/entry-requirements"
var sherpaKey = "AIzaSyDyOs9kPPkE_Dc49IDy49aHdY0y17SaErA"

var passportValidityMap = map[string]string{
	"DURATION_OF_STAY":                    "Your passport must be valid for the duration of your stay in this country.",
	"ONE_MONTH_AFTER_ENTRY":               "Your passport must be valid for one month after entering this counrty.",
	"SIX_MONTHS_AFTER_DURATION_OF_STAY":   "Your passport must be valid on entry and for six months after the duration of your stay in this country.",
	"SIX_MONTHS_AFTER_ENTRY":              "Your passport must be valid on entry and six months after the date of enrty.",
	"SIX_MONTHS_AT_ENTRY":                 "Your passport must be valid for at least six months before entering this country.",
	"THREE_MONTHS_AFTER_DURATION_OF_STAY": "Your passport must be valid on entry and for three months after the duration of your stay in this country",
	"THREE_MONTHS_AFTER_ENTRY":            "Your passport must be valid on entry and for three months after entering this country",
	"VALID_AT_ENTRY":                      "Your passport must be valid on entry",
	"THREE_MONTHS_AFTER_DEPARTURE":        "Your passport must be valid on entry and three months after your departure date.",
	"SIX_MONTHS_AFTER_DEPARTURE":          "Your passport must be valid on entry and six months after your departure date.",
}

var passportBlankpagesMap = map[string]string{
	"NOT_REQUIRED":              "You do not need to have any blank pages in your passport.",
	"ONE":                       "You need at least one blank page in your passport.",
	"ONE_PER_ENTRY":             "You need one blank page per entry.",
	"SPACE_FOR_STAMP":           "You need space for your passport to be stamped.",
	"TWO":                       "You need two blank pages in your passport.",
	"TWO_CONSECUTIVE_PER_ENTRY": "You need two consecutive blank pages in your passport",
	"TWO_PER_ENTRY":             "You need two blank pages per entry",
}

type currenciesAPIResponse struct {
	StatusCode int
	Results    map[string]interface{}
}

type currenciesConvertResponse struct {
	StatusCode int
	Results    map[string]interface{}
}

//Passport struct
type Passport struct {
	BlankPages       string          `json:"blank_pages"`
	PassportValidity string          `json:"passport_validity"`
	Textual          TextualPassport `json:"textual"`
	Currency         Currency        `json:"currency"`
}

//VaccineType struct
type VaccineType struct {
	Type      string `json:"type"`
	Condition string `json:"condition"`
}

//Vaccination struct
type Vaccination struct {
	Recommended []VaccineType `json:"recommended"`
	Required    []VaccineType `json:"required"`
	Risk        []VaccineType `json:"risk"`
}

//Currency struct
type Currency struct {
	Arrival string `json:"arrival"`
	Exit    string `json:"exit"`
}

//BlankPages struct
type BlankPages struct {
	NotRequired             string
	ONE                     string
	OnePerEntry             string
	SpaceForStamp           string
	TWO                     string
	TwoConsecutivePerEntery string
	TwoPerEntry             string
}

//PassportValidity struct
type PassportValidity struct {
	DurationOfStay                 string
	OneMonthAfterEntry             string
	SixMonthsAfterDurationOfStay   string
	SixMonthsAfterEntry            string
	SixMonthsAtEntry               string
	ThreeMonthsAfterDurationOfStay string
	ThreeMonthsAfterEntry          string
	ValidAtEntry                   string
	ThreeMonthsAfterDeparture      string
	SixMonthsAfterDepartureS       string
}

//Textual struct
type Textual struct {
	Class string   `json:"class"`
	Text  []string `json:"text"`
}

//PassportValidityTextual struct
type PassportValidityTextual struct {
	Textual Textual `json:"textual"`
}

//TextualPassport struct
type TextualPassport struct {
	Class            string                  `json:"class"`
	Text             []string                `json:"text"`
	PassportValidity PassportValidityTextual `json:"passport_validity"`
	BlankPages       Textual                 `json:"blank_pages"`
}

//Visa struct
type Visa struct {
	AllowedStay string   `json:"allowed_stay"`
	Notes       []string `json:"notes"`
	Requirement string   `json:"requirement"`
	Type        string   `json:"type"`
	Textual     Textual  `json:"textual"`
}

//VisaResponse struct
type VisaResponse struct {
	Passport    Passport    `json:"passport"`
	Visa        []Visa      `json:"visa"`
	Vaccination Vaccination `json:"vaccination"`
}

//InternalVisa struct
type InternalVisa struct {
	Visa        Visa        `json:"visa"`
	Passport    Passport    `json:"passport"`
	Vaccination Vaccination `json:"vaccination"`
}

//safetyResponse struct
type safetyResponse struct {
	Data SafetyData
}

//SafetyData struct
type SafetyData struct {
	IsoAlpha2 string         `json:"iso_alpha2" firestore:"iso_alpha2"`
	Name      string         `json:"name" firestore:"name"`
	Continent string         `json:"continent" firestore:"continent"`
	Advisory  SafetyAdvisory `json:"advisory" firestore:"advisory"`
}

//Safety struct
type Safety struct {
	Advice string  `json:"advice"`
	Rating float64 `json:"rating"`
}

//SafetyCode struct
type SafetyCode struct {
	Continent string `json:"continent"`
	Country   string `json:"country"`
}

//SafetySituation struct
type SafetySituation struct {
	Rating  string `json:"rating"`
	Sources int    `json:"sources"`
}

//SafetyAdvisory struct
type SafetyAdvisory struct {
	Score         float64 `json:"score" firestore:"score"`
	Sources       int     `json:"sources"`
	SourcesActive int     `json:"sources_active" firestore:"sources_active"`
}

//Numbers struct
type Numbers struct {
	All   []string `json:"all" firestore:"all"`
	Fixed []string `json:"fixed" firestore:"fixed"`
	GSM   []string `json:"gsm" firestore:"gsm"`
}

//EmergencyNumbers struct
type EmergencyNumbers struct {
	Ambulance               Numbers  `json:"ambulance" firestore:"ambulance"`
	Dispatch                Numbers  `json:"dispatch" firestore:"dispatch"`
	Fire                    Numbers  `json:"fire" firestore:"fire"`
	Police                  Numbers  `json:"police" firestore:"police"`
	EuropeanEmergencyNumber []string `json:"european_emergency_number"`
	Member112               bool     `json:"member_112,omitempty" firestore:"member_112"`
}

//FormatEmergencyNumbers function
func FormatEmergencyNumbers(numbers EmergencyNumbers) (e EmergencyNumbers) {
	member112 := []string{}
	if numbers.Member112 {
		member112 = []string{"112"}
	}
	e = EmergencyNumbers{
		Ambulance:               numbers.Ambulance,
		Dispatch:                numbers.Dispatch,
		Fire:                    numbers.Fire,
		Police:                  numbers.Police,
		EuropeanEmergencyNumber: member112,
	}

	return e
}

//GetCountriesCurrenciesAPI function
func GetCountriesCurrenciesAPI() (map[string]interface{}, error) {
	client := http.Client{Timeout: time.Second * 30}
	req, err := http.NewRequest(http.MethodGet, currencyDomain+"/countries?apiKey="+currencyAPIKey, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the currencies api")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the currencies api")
	}

	resp := &currenciesAPIResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("server experienced an error while parsing Currencies API response")
	}

	return resp.Results, nil

}

//ConvertCurrency function
func ConvertCurrency(to string, from string) (map[string]interface{}, error) {
	client := http.Client{Timeout: time.Second * 30}
	req, err := http.NewRequest(http.MethodGet, currencyDomain+"/convert?q="+from+"_"+to+"&compact=n&apiKey="+currencyAPIKey, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the currency converter API")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the Currency converter API")
	}

	resp := &currenciesConvertResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("server experienced an error while parsing Currency converter API response")
	}
	return resp.Results[from+"_"+to].(map[string]interface{}), nil

}

//GetVisa function
func GetVisa(destination string, citizenship string) (*VisaResponse, error) {
	client := http.Client{Timeout: time.Second * 30}
	req, err := http.NewRequest(http.MethodGet, sherpaDomain+"?key="+sherpaKey+"&citizenship="+citizenship+"&destination="+destination, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the Sherpa API")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the Sherpa API")
	}

	resp := &VisaResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("server experienced an error while parsing Sherpa API response")
	}

	return resp, nil

}

func formatPassport(passport Passport) (p *Passport) {
	blankPages := passportBlankpagesMap[passport.BlankPages]
	if len(blankPages) == 0 {
		blankPages = "To be safe make sure to have at least one blank page in your passport."
	}

	passportValidity := passportValidityMap[passport.PassportValidity]
	if len(passportValidity) == 0 {
		passportValidity = passportValidityMap["VALID_AT_ENTRY"] + " Make sure to check for additional requirements."
	}
	p = &Passport{
		PassportValidity: passportValidity,
		BlankPages:       blankPages,
	}

	return p
}

//FormatVisa function
func FormatVisa(visa VisaResponse) (v *InternalVisa) {
	var visaData Visa
	if len(visa.Visa) > 0 {
		visaData = visa.Visa[0]
	}
	v = &InternalVisa{
		Visa:        visaData,
		Passport:    *formatPassport(visa.Passport),
		Vaccination: visa.Vaccination,
	}

	return v

}

//GetSafety function
func GetSafety(countryCode string) (*SafetyData, error) {
	client := http.Client{Timeout: time.Second * 30}
	req, err := http.NewRequest(http.MethodGet, "https://www.reisewarnung.net/api?country="+countryCode, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the Safety API")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the Safety API")
	}

	resp := &safetyResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("server experienced an error while parsing Safety API response")
	}

	return &resp.Data, nil

}

//FormatSafety function
func FormatSafety(rating float64) *string {
	advice := "No safety information is available for this country."
	if rating >= 0.0 && rating < 1.0 {
		advice = "Travelling in this country is relatively safe."
	} else if rating >= 1 && rating < 2.5 {
		advice =
			"Travelling in this country is relatively safe. Higher attention is advised when traveling here due to some areas being unsafe."
	} else if rating >= 2.5 && rating < 3.5 {
		advice =
			"This country can be unsafe.  Warnings often relate to specific regions within this country. However, high attention is still advised when moving around tourist areas. Make sure not to travel to high risk areas and if you are, Trotter also recommends traveling to these areas with someone who is familiar with the culture and area."
	} else if rating >= 3.5 && rating < 4.5 {
		advice =
			"Travel to this country should be reduced to a necessary minimum and be conducted with good preparation and high attention. If you are not familiar with the area it is recommended you travel with someone who knows the area well."
	} else if rating >= 4.5 {
		advice =
			"It is unsafe to travel to this country.  Trotter advises against traveling here.  You risk high chance of danger to yourdok health and life."
	}
	return &advice
}
