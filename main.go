package main

import (
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"golang.org/x/net/html"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
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
		return
	}

	url := strings.TrimSpace(r.FormValue("url"))
	if url == "" {
		showError(w, "Empty URLs are not allowed.", http.StatusBadRequest, c)
		return
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		showError(w, "Please specify http:// or https://", http.StatusBadRequest, c)
		return
	}

	l := Link{
		URL:   url,
		Added: time.Now(),
	}

	resp, err := urlfetch.Client(c).Get(url)
	if err != nil {
		l.Title = err.Error()
	} else {
		defer resp.Body.Close()
		l.Title = parseTitle(resp.Body)
	}

	_, err = l.Save(c, uk)
	if err != nil {
		showError(w, "failed to store link", http.StatusInternalServerError, c)
		c.Errorf("failed to store link: %v", err)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func parseTitle(resp io.Reader) string {
	r := io.LimitedReader{
		R: resp,
		N: 8192,
	}

	h := html.NewTokenizer(&r)
	for {
		tt := h.Next()
		switch tt {
		case html.ErrorToken:
			return "Failed to parse page"
		case html.StartTagToken:
			tag, _ := h.TagName()
			if string(tag) == "title" {
				nt := h.Next()
				switch nt {
				case html.ErrorToken:
					return "Failed to parse title"
				case html.TextToken:
					return h.Token().Data
				}
			}
		}
	}

	return "Failed to find title"
}
