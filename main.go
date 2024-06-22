package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	fmt.Print("starting server...")
	mux.Handle("/", http.FileServer(http.Dir(".")))
	http.ListenAndServe(":8080", mux)

}
