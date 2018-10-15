// Entrypoint for API
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/asqwrd/trotter-api/firebase"
	"github.com/asqwrd/trotter-api/router"
	"github.com/gorilla/handlers"
)

func main() {
	// Get the "PORT" env variable
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	} else {
		log.Println("Running on port: " + port)
	}

	trotterFirebase.Init()

	sygicAPIKey := os.Getenv("SYGIC_API_KEY")
	if sygicAPIKey == "" {
		log.Fatal("$SYGIC_API_KEY not set")
	}

	router := router.NewRouter()

	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET"})

	// Launch server with CORS validations
	log.Fatal(http.ListenAndServe(":"+port, handlers.CORS(allowedOrigins, allowedMethods)(router)))
}
