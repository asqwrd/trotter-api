// Entrypoint for API
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/asqwrd/trotter-api/router"
	"github.com/gorilla/handlers"
)

func main() {
	// Get the "PORT" env variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
		print("$PORT not be set using dev port \n")
	} else {
		print("Running on port: " + port)
	}

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
