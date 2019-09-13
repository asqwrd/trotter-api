package router

import (
	"log"
	"net/http"

	"github.com/asqwrd/trotter-api/auth"
	"github.com/asqwrd/trotter-api/country"
	"github.com/asqwrd/trotter-api/itineraries"
	"github.com/asqwrd/trotter-api/places"
	"github.com/asqwrd/trotter-api/traxo"
	"github.com/asqwrd/trotter-api/trips"
	"github.com/asqwrd/trotter-api/users"
	"github.com/gorilla/mux"
)

// Route struct
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes type
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
		"GetCityState",
		"GET",
		"/api/explore/city_states/{countryID}/",
		country.GetCountry,
	},
	Route{
		"GetConfirmations", "GET", "/api/confirmations/", traxo.GetConfirmations,
	},
	Route{
		"GetDestination", "GET", "/api/explore/destinations/{destinationID}/", places.GetDestination,
	},
	Route{
		"GetPlaces", "GET", "/api/explore/places/", places.GetPlaces,
	},
	Route{
		"GetPark", "GET", "/api/explore/national_parks/{parkID}/", places.GetPark,
	},
	Route{
		"GetHome", "GET", "/api/explore/home/", places.GetHome,
	},
	Route{
		"GetPoi", "GET", "/api/explore/poi/{poiID}/", places.GetPoi,
	},
	Route{
		"Search", "GET", "/api/search/find/{query}/", places.Search,
	},
	Route{
		"SearchGoogle", "GET", "/api/search/google/{query}/", places.SearchGoogle,
	},
	Route{
		"RecentSearch", "GET", "/api/search/recent/", places.RecentSearch,
	},
	Route{
		"GetPopularLocations", "GET", "/api/trips/popular_locations/", places.GetPopularLocations,
	},
	Route{
		"CreateTrip", "POST", "/api/trips/create/", trips.CreateTrip,
	},
	Route{
		"AddDestination", "POST", "/api/trips/add/{tripId}", trips.AddDestination,
	},
	Route{
		"GetFlightsAndAccomodations", "GET", "/api/trips/{tripId}/flights_accomodations/", trips.GetFlightsAndAccomodations,
	},
	Route{
		"AddFlightsAndAccomodations", "POST", "/api/trips/add/flights_accomodations/{tripId}/destination/{destinationId}", trips.AddFlightsAndAccomodations,
	},
	Route{
		"DeleteFlightsAndAccomodations", "DELETE", "/api/trips/delete/flights_accomodations/{tripId}/destination/{destinationId}/detail/{detailId}", trips.DeleteFlightsAndAccomodation,
	},
	Route{
		"DeleteDestination", "DELETE", "/api/trips/delete/{tripId}/destination/{destinationId}", trips.DeleteDestination,
	},
	Route{
		"DeleteTrip", "DELETE", "/api/trips/delete/trip/{tripId}", trips.DeleteTrip,
	},
	Route{
		"UpdateTrip", "PUT", "/api/trips/update/trip/{tripId}", trips.UpdateTrip,
	},
	Route{
		"UpdateDestination", "PUT", "/api/trips/update/{tripId}/destination/{destinationId}", trips.UpdateDestination,
	},
	Route{
		"UpdateFlightsAndAccomodationTravelers", "PUT", "/api/trips/update/{tripId}/destination/{destinationId}/details/{detailId}", trips.UpdateFlightsAndAccomodationTravelers,
	},
	Route{
		"GetFlightsAndAccomodationTravelers", "GET", "/api/trips/{tripId}/travelers", trips.GetFlightsAndAccomodationTravelers,
	},
	Route{
		"AddTraveler", "POST", "/api/trips/{tripId}/travelers/add", trips.AddTraveler,
	},
	Route{
		"GetUser", "GET", "/api/users/get/{userID}", users.GetUser,
	},
	Route{
		"SearchUsers", "GET", "/api/users/search", users.SearchUsers,
	},
	Route{
		"UpdateUser", "PUT", "/api/users/update/{userID}", users.UpdateUser,
	},
	Route{
		"SaveLogin", "POST", "/api/users/login", users.SaveLogin,
	},
	Route{
		"SaveToken", "POST", "/api/users/device", users.SaveToken,
	},
	Route{
		"GetNotifications", "GET", "/api/notifications", users.GetNotifications,
	},
	Route{
		"ClearAllNotifications", "POST", "/api/notifications/clear", users.ClearAllNotifications,
	},
	Route{
		"MarkNotificationRead", "PUT", "/api/notifications/{notificationId}", users.MarkNotificationRead,
	},
	Route{
		"GetTrip", "GET", "/api/trips/get/{tripId}", trips.GetTrip,
	},
	Route{
		"GetTrips", "GET", "/api/trips/all/", trips.GetTrips,
	},
	Route{
		"GetItineraries", "GET", "/api/itineraries/all/", itineraries.GetItineraries,
	},
	Route{
		"GetItinerary", "GET", "/api/itineraries/get/{itineraryId}", itineraries.GetItinerary,
	},
	Route{
		"ChangeStartLocation", "PUT", "/api/itineraries/update/{itineraryId}/startLocation", itineraries.ChangeStartLocation,
	},
	Route{
		"GetDay", "GET", "/api/itineraries/get/{itineraryId}/day/{dayId}", itineraries.GetDay,
	},
	Route{
		"GetComments", "GET", "/api/itineraries/get/{itineraryId}/day/{dayId}/itinerary_items/{itineraryItemId}/comments", itineraries.GetComments,
	},
	Route{
		"AddComment", "POST", "/api/itineraries/{itineraryId}/day/{dayId}/itinerary_items/{itineraryItemId}/comments/add", itineraries.AddComment,
	},
	Route{
		"AddToDay", "POST", "/api/itineraries/add/{itineraryId}/day/{dayId}", itineraries.AddToDay,
	},
	Route{
		"CreateItinerary", "POST", "/api/itineraries/create", itineraries.CreateItinerary,
	},
	Route{
		"DeleteItineraryItem", "DELETE", "/api/itineraries/delete/{itineraryId}/day/{dayId}/place/{placeId}", itineraries.DeleteItineraryItem,
	},
	Route{
		"TestNotification", "GET", "/api/notification/test", itineraries.TestNotification,
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
			Handler(auth.BasicAuthMiddleware(handler))
	}
	return router
}
