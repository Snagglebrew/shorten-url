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

func AuthorizePublicUser(secretKey string) bool {
	r := database.CreateClient(2)
	defer r.Close()
	return r.SIsMember(database.Ctx, "users:public", secretKey).Val()
}
