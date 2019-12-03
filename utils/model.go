package utils

import (
	"cloud.google.com/go/firestore"
	"github.com/asqwrd/trotter-api/types"
	"golang.org/x/net/context"
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

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func Filter(s []string, e string) []string {
	var output []string
	for _, a := range s {
		if a != e {
			output = append(output, a)
		}
	}
	return output
}

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
