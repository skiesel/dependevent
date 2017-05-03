package dependevent

import (
	"google.golang.org/appengine"
	"google.golang.org/appengine/user"
	"net/http"
)

func dashboard(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)

	if u == nil {
		showLogin(w, r, c)
		return
	}

	account := getAccountByEmail(u.Email, c)
	account.populateEvents(c)

	err := pages.ExecuteTemplate(w, "dashboard.template", account)
	if err != nil {
		showError(w, http.StatusInternalServerError, err, c)
	}
}
