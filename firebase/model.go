package trotterFirebase

import (
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

var firebaseClient firestore.Client

func Init() {
	// Use a service account
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := getContext()
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()
	setClient(*client)
}

func getContext() context.Context {
	return context.Background()
}

func setClient(client firestore.Client) {
	firebaseClient = client
}

func getClient() *firestore.Client {
	return &firebaseClient
}

func GetCollection(collection string) *firestore.DocumentIterator {
	ctx := getContext()
	client := getClient()
	iter := client.Collection(collection).Documents(ctx)
	return iter
}
