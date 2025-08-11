package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
)

//go:embed configs/templates/index.html.templ
var indexPage embed.FS

func handleRoot(w http.ResponseWriter, r *http.Request) {

	var indexTempl bytes.Buffer

	f, _ := indexPage.ReadFile("configs/templates/index.html.templ")

	if tpl, err := template.New("").Parse(string(f)); err == nil {

		templmu.RLock()
		tpl.Execute(&indexTempl, templ)
		templmu.RUnlock()

	}

	w.Write(indexTempl.Bytes())
}

func handleServerlist(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(templ)

}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		fmt.Printf("%T\n%+v\n%#v\n\n", next, next, next)

		next.ServeHTTP(w, r)

		fmt.Println(next)
	})
}
