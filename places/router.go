package places

import (
	"log"
	"net/http"

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
		GetContinent,
	},
	Route{
		"GetCountry",
		"GET",
		"/api/explore/country/{countryID}/",
		GetCountry,
	},
	Route{
		"GetCity", "GET", "/api/explore/cities/{cityID}/", GetCity,
	},
	Route{
		"GetHome", "GET", "/api/explore/home/", GetHome,
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
