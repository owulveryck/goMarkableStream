package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
)

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	decoded, err := base64.StdEncoding.DecodeString(favicon)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("content-type", "image/x-icon")
	fmt.Fprint(w, string(decoded))
}
