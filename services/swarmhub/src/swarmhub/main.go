package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/api"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/jwt"

	"github.com/julienschmidt/httprouter"
)

var (
	htmlDir          string
	LdapAddress      string
	UsersFromFile    string
	JwtFile          string
	UserFileLocation string
	tlsCertFileLoc   string
	tlsKeyFileLoc    string
)

type HomeData struct {
	User    string
	Message string
}

func HomePage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	file, err := ioutil.ReadFile(htmlDir + "/index.html")
	if err != nil {
		log.Print("error reading index file: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(file)
}

func LogoutPost(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	c := http.Cookie{
		Name:    "Authorization",
		Path:    "/",
		Value:   "",
		MaxAge:  -10,
		Expires: time.Unix(0, 0),
	}
	http.SetCookie(w, &c)
	fmt.Println("Operation Delete cookie completed")
	data := &HomeData{"", ""}
	t, err := template.ParseFiles(htmlDir+"/logout.html", htmlDir+"/base.html")
	if err != nil {
		log.Print("template parsing error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Print("template executing error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func LoginPageMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, message string) {
	user := jwt.TokenAudienceFromRequest(r)
	data := &HomeData{user, message}

	t, err := template.ParseFiles(htmlDir+"/login.html", htmlDir+"/base.html")
	if err != nil {
		log.Print("template parsing error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Print("template executing error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func LoginPageGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user := jwt.TokenAudienceFromRequest(r)
	data := &HomeData{user, ""}

	t, err := template.ParseFiles(htmlDir+"/login.html", htmlDir+"/base.html")
	if err != nil {
		log.Print("template parsing error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Print("template executing error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func LoginPagePost(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Println("Parse form")
	r.ParseForm()

	fmt.Println("Extracting username and password")

	username := r.Form["username"][0]
	password := r.Form["password"][0]

	fmt.Println("checking username password. Username is: ", username)

	tokenString, err := login(username, password)
	if err != nil {
		fmt.Printf("login error for user %v: %v\n", username, err)
		w.WriteHeader(http.StatusSeeOther)
		LoginPageMessage(w, r, ps, "Unable to login, make sure you are using correct credentials.")
		return
	}

	cookie := &http.Cookie{
		Name:   "Authorization",
		Value:  tokenString,
		Path:   "/",
		MaxAge: 86400,
	}

	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func TokenAuth(handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		cookie, err := r.Cookie("Authorization")
		if err != nil {
			fmt.Println(err.Error())
			LoginPageGet(w, r, nil)
			return
		}

		ok, err := jwt.ValidateToken(cookie.Value)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		if ok {
			handler(w, r, ps)
			return
		}

		LoginPageGet(w, r, nil)

		//w.Write([]byte("Unauthorized Please Re-Login.\n"))
		return
	}
}

func main() {
	ConfigSet()
	api.StartNats(Registry)

	router := httprouter.New()

	api.SetRouterPaths(router)
	router.ServeFiles("/static/*filepath", http.Dir("/var/www/swarmhub/static/"))
	router.GET("/", TokenAuth(HomePage))
	router.GET("/tests", TokenAuth(HomePage))
	router.GET("/tests/:id", TokenAuth(HomePage))
	router.GET("/tests/:id/logs", TokenAuth(HomePage))
	router.GET("/grids", TokenAuth(HomePage))
	router.GET("/grids/:id", TokenAuth(HomePage))
	router.GET("/grids/:id/logs", TokenAuth(HomePage))
	router.GET("/login", LoginPageGet)
	router.POST("/login", LoginPagePost)
	router.POST("/logout", LogoutPost)

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)
	go api.Shutdown(signalChan)

	fmt.Println("started up")
	err := http.ListenAndServeTLS(":8443", tlsCertFileLoc, tlsKeyFileLoc, router)
	if err != nil {
		fmt.Println(err.Error())
	}
}
