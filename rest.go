package main

import (
	"bytes"
	"net/http"
)

func apiHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(bytes.NewBufferString("Test").Bytes())
}
