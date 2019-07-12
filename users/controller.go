package users

import (
	//"sort"
	"encoding/json"
	"fmt"
	"net/http"

	//"github.com/asqwrd/trotter-api/triposo"
	"net/url"

	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/types"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// SaveLogin function
func SaveLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var user types.User
	err := decoder.Decode(&user)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	//Check User
	docSnap, _ := client.Collection("users").Doc(user.UID).Get(ctx)
	if docSnap.Exists() == false {

		_, errUserCreate := client.Collection("users").Doc(user.UID).Set(ctx, user)
		if errUserCreate != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(errUserCreate)
			response.WriteErrorResponse(w, errUserCreate)
			return
		}
	} else {
		response.Write(w, map[string]interface{}{
			"success": true,
			"exists":  true,
		}, http.StatusOK)
		return
	}

	fmt.Println("User Added")

	userData := map[string]interface{}{
		"success": true,
		"exists":  false,
	}

	response.Write(w, userData, http.StatusOK)
	return
}

// SaveToken function
func SaveToken(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var token types.Token
	err := decoder.Decode(&token)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	//Check User
	docSnap, _ := client.Collection("users").Doc(token.UID).Collection("devices").Doc(token.Token).Get(ctx)
	if docSnap.Exists() == false {

		_, errDeviceCreate := client.Collection("users").Doc(token.UID).Collection("devices").Doc(token.Token).Set(ctx, token)
		if errDeviceCreate != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(errDeviceCreate)
			response.WriteErrorResponse(w, errDeviceCreate)
			return
		}
	} else {
		response.Write(w, map[string]interface{}{
			"success": true,
			"exists":  true,
		}, http.StatusOK)
		return
	}

	fmt.Println("Device Added")

	userData := map[string]interface{}{
		"success": true,
		"exists":  false,
	}

	response.Write(w, userData, http.StatusOK)
	return
}

// GetNotifications function
func GetNotifications(w http.ResponseWriter, r *http.Request) {
	var q *url.Values
	args := r.URL.Query()
	q = &args
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	uuid := q.Get("user_id")

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	notifications := []types.Notification{}
	iter := client.Collection("users").Doc(uuid).Collection("notifications").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		var notification types.Notification
		doc.DataTo(&notification)
		notifications = append(notifications, notification)
	}

	fmt.Println("Got Notifications")

	userData := map[string]interface{}{
		"success":       true,
		"notifications": notifications,
	}

	response.Write(w, userData, http.StatusOK)
	return
}