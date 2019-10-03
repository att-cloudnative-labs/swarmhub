package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/dgrijalva/jwt-go"
)

var (
	tlsCertfileLoc = os.Getenv("TLS_CERT_FILE_LOC")
	tlsKeyFileLoc = os.Getenv("TLS_KEY_FILE_LOC")
)

func main() {
	if tlsCertfileLoc == "" || tlsKeyFileLoc == "" {
		tlsCertfileLoc = "server.crt"
		tlsKeyFileLoc = "server.key"
	}
	serverLocust := http.NewServeMux()
	serverLocust.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/login" {
			login(w, r)
			return
		}

		if r.URL.Path != "/metrics" {
			valid, _ := validate(r)
			if !valid {
				w.WriteHeader(401)
				w.Write([]byte("Unauthorized.\n"))
				return
			}
		}

		target, err := url.Parse("http://localhost:8089")
		if err != nil {
			fmt.Println(err.Error())
		}

		director := func(req *http.Request) {
			req.URL.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.Host = target.Host
		}

		proxy := &httputil.ReverseProxy{Director: director}

		proxy.ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServeTLS(":443", tlsCertfileLoc, tlsKeyFileLoc, serverLocust))
}

func login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	//GET displays the upload form.
	case "POST":
		fmt.Println("Parse form")
		r.ParseForm()

		fmt.Println("Extracting username and password")

		authToken := r.Form["authToken"][0]

		cookie := &http.Cookie{
			Name:   "Authorization",
			Value:  authToken,
			Path:   "/",
			MaxAge: 86400,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

}

func validate(r *http.Request) (bool, error) {
	cookie, err := r.Cookie("Authorization")
	if err != nil {
		return false, err
	}

	ok, err := validateToken(cookie.Value)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func validateToken(tokenString string) (bool, error) {
	signingKey, err := ioutil.ReadFile("jwt")
	if err != nil {
		return false, err
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return signingKey, nil
	})
	if err != nil {
		return false, err
	}

	if token.Valid {
		return true, err
	}
	return false, err
}
