package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	fmt.Print("starting server...")

	// logoPath := "/assets/logo.png"
	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	http.ListenAndServe(":8080", mux)

}
