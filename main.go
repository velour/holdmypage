package main

import (
	"net/http"
	"text/template"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

var pages *template.Template

func init() {
	pages = template.Must(template.ParseGlob("pages/*.html"))
	http.HandleFunc("/", showIndex)
}

func showIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	c := appengine.NewContext(r)

	u := user.Current(c)
	if u == nil {
		showLogin(w, c)
		return
	}

	//TODO: we're using google accounts for now,
	// but this may need to change to something more complex if we change.
	uk := datastore.NewKey(c, "User", u.ID, 0, nil)
	var us User
	err := datastore.Get(c, uk, &us)
	if err != nil && err != datastore.ErrNoSuchEntity {
		showError(w, "Ask Steve to look.", http.StatusInternalServerError, c)
		c.Errorf("failed to retrieve user %q: %v", u.String(), err)
		return
	}
	if err == datastore.ErrNoSuchEntity {
		u.Email = u.Email
	}

	links := []Link{}
	lks, err := datastore.NewQuery("Link").Ancestor(uk).GetAll(c, &links)
	if err != nil {
		showError(w, "Ask Scott to look.", http.StatusInternalServerError, c)
		c.Errorf("failed to retrieve user's links %q: %v", u.String(), err)
		return
	}

	x := struct{
		User User
		Links []Link
		LinkKeys []*datastore.Key
	}{
		User: us,
		Links: links,
		LinkKeys: lks,
	}
	err = pages.ExecuteTemplate(w, "index.html", x)
	if err != nil {
		http.Error(w, "Failed to render the page.", http.StatusInternalServerError)
		c.Errorf("failed to render index: %v", err)
		return
	}
}

func showLogin(w http.ResponseWriter, c appengine.Context) {
	login, err := user.LoginURL(c, "/")
	if err != nil {
		showError(w, "Ask Steve.", http.StatusInternalServerError, c)
		c.Errorf("Failed to get login url: %v", err)
		return
	}

	err = pages.ExecuteTemplate(w, "login.html", login)
	if err != nil {
		http.Error(w, "Failed to render the page.", http.StatusInternalServerError)
		c.Errorf("failed to render login: %v", err)
		return
	}
}

func showError(w http.ResponseWriter, msg string, status int, c appengine.Context) {
	err := pages.ExecuteTemplate(w, "oops.html", map[string]string{
		"Message": msg,
		"Status": http.StatusText(status),
	})
	if err != nil {
		http.Error(w, "Failed to render the page.", http.StatusInternalServerError)
		c.Errorf("failed to render error: %v", err)
		return
	}
}

type User struct {
	Email string
	Tags []string
}

type Link struct {
	URL string
	Title string
	Tags []string
}
