package users

import (
	//"sort"
	"encoding/json"
	"fmt"
	"net/http"

	//"github.com/asqwrd/trotter-api/triposo"
	"net/url"

	"github.com/algolia/algoliasearch-client-go/algolia/search"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/types"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func SearchUsers(w http.ResponseWriter, r *http.Request) {

	var users []types.User
	var q *url.Values
	args := r.URL.Query()
	q = &args

	client := search.NewClient(types.Algolia_APPID, types.Algolia_ADMINKEY)
	index := client.InitIndex("users")

	res, err := index.Search(q.Get("query"))
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}

	err = res.UnmarshalHits(&users)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}

	response.Write(w, map[string]interface{}{
		"results": users,
	}, http.StatusOK)

}

func GetUser(w http.ResponseWriter, r *http.Request) {
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	userID := mux.Vars(r)["userID"]
	var user types.User

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()
	docSnap, errGet := client.Collection("users").Doc(userID).Get(ctx)
	if errGet != nil {
		fmt.Println(errGet)
		response.WriteErrorResponse(w, errGet)
		return
	}
	errDataTo := docSnap.DataTo(&user)
	if errDataTo != nil {
		fmt.Println(errDataTo)
		response.WriteErrorResponse(w, errDataTo)
		return
	}
	response.Write(w, map[string]interface{}{
		"user": user,
	}, http.StatusOK)

}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	decoder := json.NewDecoder(r.Body)
	userID := mux.Vars(r)["userID"]
	var user map[string]interface{}
	err := decoder.Decode(&user)
	if err != nil {
		fmt.Println(err)
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()
	_, errSet := client.Collection("users").Doc(userID).Set(ctx, user, firestore.MergeAll)
	if errSet != nil {
		fmt.Println(errSet)
		response.WriteErrorResponse(w, errSet)
		return
	}
	response.Write(w, map[string]interface{}{
		"success": true,
	}, http.StatusOK)
	return

}

// SaveLogin function
func SaveLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var user types.User
	err := decoder.Decode(&user)
	if err != nil {
		fmt.Println(err)
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	//Check User
	docSnap, _ := client.Collection("users").Doc(user.UID).Get(ctx)
	if docSnap.Exists() == false {
		user.NotificationsOn = true
		user.Country = "US"
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	_, errDeviceCreate := client.Collection("users").Doc(token.UID).Collection("devices").Doc(token.DeviceID).Set(ctx, token)
	if errDeviceCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDeviceCreate)
		response.WriteErrorResponse(w, errDeviceCreate)
		return
	}

	fmt.Println("Device token updated")

	userData := map[string]interface{}{
		"success": true,
		"exists":  false,
	}

	response.Write(w, userData, http.StatusOK)
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	notifications := []types.Notification{}
	iter := client.Collection("users").Doc(uuid).Collection("notifications").Where("read", "==", false).OrderBy("created_at", firestore.Desc).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
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

// MarkNotificationRead function
func MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	var q *url.Values
	args := r.URL.Query()
	q = &args
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	fmt.Println("Start")

	notificationID := mux.Vars(r)["notificationId"]
	fmt.Println(notificationID)

	uuid := q.Get("user_id")

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	_, errMark := client.Collection("users").Doc(uuid).Collection("notifications").Doc(notificationID).Set(ctx, map[string]interface{}{
		"read": true,
	}, firestore.MergeAll)

	if errMark != nil {
		fmt.Println(errMark)
		response.WriteErrorResponse(w, errMark)
		return
	}
	fmt.Println("Here")

	notifications := []types.Notification{}
	iter := client.Collection("users").Doc(uuid).Collection("notifications").Where("read", "==", false).OrderBy("created_at", firestore.Desc).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
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

// ClearAllNotifications function
func ClearAllNotifications(w http.ResponseWriter, r *http.Request) {
	var q *url.Values
	args := r.URL.Query()
	q = &args
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	fmt.Println("Start")

	uid := q.Get("user_id")

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	iter := client.Collection("users").Doc(uid).Collection("notifications").Where("read", "==", false).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}
		var notification types.Notification
		doc.DataTo(&notification)
		_, errMark := client.Collection("users").Doc(uid).Collection("notifications").Doc(notification.ID).Set(ctx, map[string]interface{}{
			"read": true,
		}, firestore.MergeAll)

		if errMark != nil {
			fmt.Println(errMark)
			response.WriteErrorResponse(w, errMark)
			return
		}
	}

	fmt.Println("Cleared Notifications")

	userData := map[string]interface{}{
		"success":       true,
		"notifications": []interface{}{},
	}

	response.Write(w, userData, http.StatusOK)
	return
}
