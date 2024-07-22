package helpers

import (
	"os"
	"strings"

	"github.com/snagglebrew/shorten-url/database"
)

func EnforceHTTP(url string) string {
	if url[:4] != "http" {
		return "http://" + url
	}
	return url
}

func RemoveDomainError(url string) bool {
	newURL := strings.Replace(url, "http://", "", 1)
	newURL = strings.Replace(newURL, "https://", "", 1)
	newURL = strings.Replace(newURL, "www.", "", 1)
	newURL = strings.Split(newURL, "/")[0]
	return newURL != os.Getenv("DOMAIN")
}

// AuthorizePublicUser checks if the secret key is in the public user set
// r := database.CreateClient(0) Public user authorization
func AuthorizePublicUser(username string, secretKey string) bool {
	r := database.CreateClient(0)
	defer r.Close()
	if r.SIsMember(database.Ctx, "users:public", username).Val() {
		return r.HGet(database.Ctx, username, "Secret").Val() == secretKey
	}
	return false
}
