package utils

import (
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/asqwrd/trotter-api/types"
	"golang.org/x/net/context"
	"gopkg.in/maddevsio/fcm.v1"
)

// Block Type
type Block struct {
	Try     func()
	Catch   func(Exception)
	Finally func()
}

// Exception type
type Exception interface{}

// Throw function
func Throw(up Exception) {
	panic(up)
}

// Do function
func (tcf Block) Do() {
	if tcf.Finally != nil {

		defer tcf.Finally()
	}
	if tcf.Catch != nil {
		defer func() {
			if r := recover(); r != nil {
				tcf.Catch(r)
			}
		}()
	}
	tcf.Try()
}

// CheckFirestoreQueryResults function
func CheckFirestoreQueryResults(ctx context.Context, query firestore.Query) bool {

	defer func() bool {
		if err := recover(); err != nil {
			return false
		}
		return true
	}()
	query.Documents(ctx)

	return true
}

// FindInTripGroup function
func FindInTripGroup(group []interface{}, queryUser types.User) bool {
	for _, s := range group {
		if s == queryUser.UID {
			return true
		}
	}

	return false
}

//Contains function
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

//Filter function
func Filter(s []string, e string) []string {
	var output []string
	for _, a := range s {
		if a != e {
			output = append(output, a)
		}
	}
	return output
}

//UniqueUserSlice function
func UniqueUserSlice(userSlice []types.User) []types.User {
	keys := make(map[string]bool)
	list := []types.User{}
	for _, entry := range userSlice {
		if _, value := keys[entry.UID]; !value {
			keys[entry.UID] = true
			list = append(list, entry)
		}
	}
	return list
}

//UniqueDestinationsSlice function
func UniqueDestinationsSlice(destinationSlice []map[string]interface{}) []map[string]interface{} {
	keys := make(map[string]bool)
	list := []map[string]interface{}{}
	for _, entry := range destinationSlice {
		key := entry["destination"].(types.Destination).DestinationID
		if _, value := keys[key]; !value {
			keys[key] = true
			list = append(list, entry)
		}
	}
	return list
}

//SendNotification function
func SendNotification(navigateData map[string]interface{}, msg string, actingUser types.User, notificationType string, key string, tokens []string) {
	c := fcm.NewFCM(types.SERVER_KEY)
	if len(tokens) > 0 {

		data := map[string]interface{}{
			"focus":            "trips",
			"click_action":     "FLUTTER_NOTIFICATION_CLICK",
			"type":             notificationType,
			"notificationData": navigateData,
			"user":             actingUser,
			"msg":              msg,
		}

		notification, err := c.Send(fcm.Message{
			Data:             data,
			RegistrationIDs:  tokens,
			CollapseKey:      key,
			ContentAvailable: true,
			Priority:         fcm.PriorityNormal,
			Notification: fcm.Notification{
				Title:       key,
				Body:        msg,
				ClickAction: "FLUTTER_NOTIFICATION_CLICK",
				//Badge: user.PhotoURL,
			},
		})
		if err != nil {
			fmt.Println("Notification send err")
			fmt.Println(err)
			//response.WriteErrorResponse(w, err)
		}
		fmt.Println("Status Code   :", notification.StatusCode)
		fmt.Println("Success       :", notification.Success)
		fmt.Println("Fail          :", notification.Fail)
		fmt.Println("Canonical_ids :", notification.CanonicalIDs)
		fmt.Println("Topic MsgId   :", notification.MsgID)
	}
}
