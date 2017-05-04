package dependevent

import (
	"fmt"
  "io/ioutil"
  "log"
  "net/http"

	"google.golang.org/appengine/user"
	"google.golang.org/appengine"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
  "google.golang.org/api/calendar/v3"
)

var (
	config *oauth2.Config
)

func init() {
	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	config, err = google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
}

func startAuthProcess(w http.ResponseWriter, r *http.Request, intendedDestination string) {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)

	account := getAccountByEmail(u.Email, ctx)
	account.IntendedDestination = intendedDestination
	account.saveInDS(ctx)

	redirectURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOnline)
	http.Redirect(w, r, redirectURL, 303)
}

func gotAuthResponse(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)

	if r.URL.Query()["code"] == nil || r.URL.Query()["code"][0] == "" {
		fmt.Fprintf(w, "token is absent")
		log.Fatalf("token is absent")
	}

	token, err := config.Exchange(ctx, r.URL.Query()["code"][0])
  if err != nil {
		fmt.Fprintf(w, "Unable to retrieve token %v", err)
    log.Fatalf("Unable to retrieve token %v", err)
		return
  }

	account := getAccountByEmail(u.Email, ctx)
	destination := account.IntendedDestination
	account.IntendedDestination = ""

	account.OAuthToken = *token

	account.saveInDS(ctx)

	http.Redirect(w, r, destination, 303)
}
