package dependevent

import (
	"appengine"
	"appengine/user"
	"fmt"
	"net/http"
)

func index(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)

	if u == nil {
		showLogin(w, r, c)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func logout(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)

	if u != nil {
		showLogout(w, r, c)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func showLogin(w http.ResponseWriter, r *http.Request, c appengine.Context) {
	login, err := user.LoginURL(c, "/")
	if err != nil {
		showError(w, http.StatusInternalServerError, err, c)
		return
	}

	http.Redirect(w, r, login, http.StatusFound)
}

func showLogout(w http.ResponseWriter, r *http.Request, c appengine.Context) {
	logout, err := user.LogoutURL(c, "/")
	if err != nil {
		showError(w, http.StatusInternalServerError, err, c)
		return
	}

	http.Redirect(w, r, logout, http.StatusFound)
}

func showError(w http.ResponseWriter, status int, err error, c appengine.Context) {
	c.Errorf("%v", err)
	fmt.Fprintf(w, "Perhaps you've had one too many...")
}
