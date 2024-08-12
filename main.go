package main

import (
	"Mars/auth"
	"Mars/backend"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func main() {
	if len(os.Args) > 1 {
		fmt.Println("hello")
		os.Exit(0)
	}
	http.HandleFunc("/ws", backend.Handler)

	// serve static files
	frontendDir := "frontend"
	staticDir := filepath.Join(frontendDir, "static")
	stylesDir := filepath.Join(frontendDir, "styles")
	jsDir := filepath.Join(frontendDir, "js")
	templatesDir := filepath.Join(frontendDir, "templates")

	fsStatic := http.FileServer(http.Dir(staticDir))
	http.Handle("/static/", noCache(http.StripPrefix("/static/", fsStatic)))

	fsConf := http.FileServer(http.Dir("conf"))
	http.Handle("/conf/", noCache(http.StripPrefix("/conf/", fsConf)))

	fsStyles := http.FileServer(http.Dir(stylesDir))
	http.Handle("/styles/", noCache(http.StripPrefix("/styles/", fsStyles)))

	fsJs := http.FileServer(http.Dir(jsDir))
	http.Handle("/js/", noCache(http.StripPrefix("/js/", fsJs)))

	fsTemplates := http.FileServer(http.Dir(templatesDir))
	http.Handle("/", noCache(fsTemplates))

	MarsPort := auth.GetFromConf("Mars_PORT")
	log.Println("[MAIN] - starting server on :", MarsPort)
	port := fmt.Sprintf(":%s", MarsPort)
	log.Fatal(http.ListenAndServe(port, nil))
}
