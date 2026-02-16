package main

import (
	"fmt"
	"net/http"
	"os"

	"streamweb/api/internal/httpapi"
	"streamweb/api/internal/service"
	"streamweb/api/internal/store"
)

func main() {
	st := store.NewMemoryStore()
	secret := os.Getenv("TOKEN_SECRET")
	svc := service.New(st, secret)
	srv := httpapi.NewServer(svc)

	mux := http.NewServeMux()
	srv.Register(mux)

	fmt.Println("API listening on :8080")
	_ = http.ListenAndServe(":8080", mux)
}
