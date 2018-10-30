package router

import (
	"log"
	"net/http"

	"github.com/asqwrd/trotter-api/country"
	"github.com/asqwrd/trotter-api/places"
	"github.com/gorilla/mux"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	Route{
		"GetContinent",
		"GET",
		"/api/explore/continent/{continentID}/",
		places.GetContinent,
	},
	Route{
		"GetCountry",
		"GET",
		"/api/explore/countries/{countryID}/",
		country.GetCountry,
	},
	Route{
		"GetCity", "GET", "/api/explore/cities/{cityID}/", places.GetCity,
	},
	Route{
		"GetPark", "GET", "/api/explore/national_parks/{parkID}/", places.GetPark,
	},
	Route{
		"GetHome", "GET", "/api/explore/home/", places.GetHome,
	},
	Route{
		"GetPoi", "GET", "/api/explore/poi/{poiID}", places.GetPoi,
	},
	Route{
		"Search", "GET", "/api/explore/search/{query}", places.Search,
	},
}

// NewRouter configures a new router to the API
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		log.Println(route.Name)
		handler = route.HandlerFunc

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}
	return router
}
