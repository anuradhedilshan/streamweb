package main

import (
	"fmt"
	"net/http"

	"streamweb/api/internal/httpapi"
	"streamweb/api/internal/service"
	"streamweb/api/internal/store"
)

func main() {
	st := store.NewMemoryStore()
	svc := service.New(st)
	srv := httpapi.NewServer(svc)

	mux := http.NewServeMux()
	srv.Register(mux)

	fmt.Println("API listening on :8080")
	_ = http.ListenAndServe(":8080", mux)
}
