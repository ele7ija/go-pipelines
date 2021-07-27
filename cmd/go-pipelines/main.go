package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	pipeApi "github.com/ele7ija/go-pipelines/internal"
	"github.com/ele7ija/go-pipelines/workers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"net/http"
	"strconv"
	"time"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "go-pipelines"
	password = "go-pipelines"
	dbname   = "go-pipelines"
)


// TODO rewrite stuff using resource and store terminology https://github.com/dhax/go-base/blob/c3809c7cabc5d64b30d9a413897e261a2e3e819c/api/app/account.go#L44

func main() {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected to DB!")

	r.Mount("/images", imagesRouter(db))

	if err := http.ListenAndServe(":3333", r); err != nil {
		return
	}
}

func imagesRouter(db *sql.DB) http.Handler {

	r := chi.NewRouter()
	r.Use(UserOnly)
	r.Get("/", getAllImages(db))
	r.Get("/{imageId}", getImage(db))

	r.Post("/", createImages(db))
	return r
}

func createImages(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {

	//imagesService := workers.NewImageService(db)
	return func(w http.ResponseWriter, r *http.Request) {

		w.Write([]byte(fmt.Sprintf("created")))
	}
}


func getAllImages(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {

	imagesService := workers.NewImageService(db)
	return func(w http.ResponseWriter, r *http.Request) {

		pipeline := workers.GetAllImagesPipeline(imagesService)

		images, errors, err := imagesService.GetAllMetadata(r.Context())
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("errored while getting images metadata"))
			return
		}
		startingItems := make(chan pipeApi.Item)
		go func() {
			for img := range images {
				startingItems <- pipeApi.NewGenericItem(img)
			}
			close(startingItems)
		}()
		go func() {
			for err := range errors {
				fmt.Println("Error in the GetAllMetadata: ", err)
			}
		}()

		items, errors := pipeline.Filter(r.Context(), startingItems)
		go func() {
			for err := range errors {
				fmt.Println("Error in the GetImagePipeline: ", err)
			}
		}()

		for item := range items {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(item.GetPart(0).(workers.Image)); err != nil {
				w.WriteHeader(500)
			}
		}
	}
}

func getImage(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {

	imagesService := workers.NewImageService(db)
	return func(w http.ResponseWriter, r *http.Request) {

		pipeline := workers.GetImagePipeline(imagesService)
		imageId, err := strconv.Atoi(chi.URLParam(r, "imageId"))
		if err != nil {
			w.WriteHeader(404)
			w.Write([]byte("image id not an integer"))
			return
		}

		ch := make(chan pipeApi.Item, 1)
		ch <- pipeApi.NewGenericItem(imageId)
		items, errors := pipeline.Filter(r.Context(), ch)
		go func() {
			for err := range errors {
				fmt.Println("Error in the GetImagePipeline: ", err)
			}
		}()

		for item := range items {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(item.GetPart(0).(workers.Image)); err != nil {
				w.WriteHeader(500)
			}
		}
	}
}

// UserOnly inserts the userId into context from jwt
func UserOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO check for jwt
		ctx := context.WithValue(r.Context(), "userId", 1)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}