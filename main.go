package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"slack-shiba-bot/handlers"
	"syscall"

	chi "github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	log.Printf("Starting the bot...")

	signal.Ignore(syscall.SIGPIPE)
	// load the .env

	godotenv.Load()

	// start the chi router

	r := chi.NewRouter()

	// add the routes

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi ^-^"))
	})

	r.Post("/slack/today", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleTodayCommand(w, r)
	})

	// start the server
	var port = "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Printf("Listening on port %s...", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}

}
