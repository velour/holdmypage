package main

import (
	"net/http"
	"text/template"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

var pages *template.Template

func init() {
	pages = template.Must(template.ParseGlob("pages/*.html"))
	http.HandleFunc("/", showIndex)
	http.HandleFunc("/add", addLink)
}

func showIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	c := appengine.NewContext(r)

	u := user.Current(c)
	if u == nil {
		showLogin(w, c)
		return
	}

	us, uk, err := getUser(c)
	if err != nil {
		showError(w, "Ask Steve to look.", http.StatusInternalServerError, c)
		c.Errorf("failed to retrieve user %q: %v", u.String(), err)
	}

	links := []Link{}
	lks, err := datastore.NewQuery("Link").
		Ancestor(uk).
		Order("-Added").
		GetAll(c, &links)
	if err != nil {
		showError(w, "Ask Scott to look.", http.StatusInternalServerError, c)
		c.Errorf("failed to retrieve user's links %q: %v", u.String(), err)
		return
	}

	x := struct {
		User     User
		Links    []Link
		LinkKeys []*datastore.Key
	}{
		User:     us,
		Links:    links,
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
		"Status":  http.StatusText(status),
	})
	if err != nil {
		http.Error(w, "Failed to render the page.", http.StatusInternalServerError)
		c.Errorf("failed to render error: %v", err)
		return
	}
}

func addLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	c := appengine.NewContext(r)
	u := user.Current(c)

	_, uk, err := getUser(c)
	if err != nil {
		showError(w, "failed to retrieve user", http.StatusInternalServerError, c)
		c.Errorf("failed to retrieve user %q: %v", u.String(), err)
	}

	l := Link{
		URL:   r.FormValue("url"),
		Added: time.Now(),
	}
	_, err = l.Save(c, uk)
	if err != nil {
		showError(w, "failed to store link", http.StatusInternalServerError, c)
		c.Errorf("failed to store link: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
