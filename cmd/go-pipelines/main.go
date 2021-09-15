package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ele7ija/go-pipelines/workers"
	pipe "github.com/ele7ija/pipeline"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	metrics "github.com/tevjef/go-runtime-metrics"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	host         = "localhost"
	port         = 5432
	user         = "go-pipelines"
	password     = "go-pipelines"
	dbname       = "go-pipelines"
	MaxOpenConns = 40
)

func main() {

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.999",
	})
	log.SetLevel(log.DebugLevel)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(600 * time.Second))
	r.Use(middleware.SetHeader("Access-Control-Allow-Origin", "*"))

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(MaxOpenConns)
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	log.Info("Successfully connected to DB!")

	r.Mount("/api/images", imagesRouter(db))

	fs := http.FileServer(http.Dir("static"))
	r.Handle("/*", http.StripPrefix("", fs))

	// Collect performance stats
	go func() {
		conf := &metrics.Config{
			Host:               "localhost:8086",
			Database:           "stats",
			Username:           "go-pipelines",
			Password:           "go-pipelines",
			CollectionInterval: time.Second,
		}
		if err := metrics.RunCollector(conf); err != nil {
			log.Errorf("An error happened while sending performance stats to InfluxDB: %s", err)
		}
	}()

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

	imagesService := workers.NewImageService(db)
	pipeline := workers.MakeCreateImagesPipeline(imagesService)

	return func(w http.ResponseWriter, r *http.Request) {

		err := r.ParseMultipartForm(2 << 30) // 1GB
		if err != nil {
			log.Errorf("ParseMultiPartForm error: %s", err)
			w.WriteHeader(400)
			return
		}
		fhs := r.MultipartForm.File["images"]

		// asynchronously add starting items
		startingItems := make(chan pipe.Item, len(fhs))
		go func() {
			wg := sync.WaitGroup{}
			wg.Add(len(fhs))
			for _, fh := range fhs {
				startingItems <- fh
				wg.Done()
			}
			wg.Wait()
			close(startingItems)
		}()

		errors := make(chan error, len(fhs))
		started := time.Now()
		items := pipeline.Filter(r.Context(), startingItems, errors)
		go func() {
			for err := range errors {
				log.Errorf("Error in the CreateImagesPipeline: %v", err)
			}
		}()

		// Send response
		w.Header().Set("Content-Type", "application/json")
		counter := 0
		w.Write([]byte("{\"images\": ["))
		for item := range items {
			log.Debugf("Sending image no: %d", counter)
			counter++
			img := item.(*workers.ImageBase64)
			err := json.NewEncoder(w).Encode(img)
			if err != nil {
				w.WriteHeader(500)
			}
			w.Write([]byte(","))
		}

		w.Write([]byte("\"void\"]}"))
		pipeline.FilteringNumber++
		pipeline.FilteringDuration += time.Since(started)
		close(errors)
	}
}

func getAllImages(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {

	imagesService := workers.NewImageService(db)
	pipeline := workers.MakeGetAllImagesPipeline(imagesService)

	return func(w http.ResponseWriter, r *http.Request) {

		images, errors, err := imagesService.GetAllMetadata(r.Context())
		if err != nil {
			log.Errorf("%v", err)
			w.WriteHeader(500)
			w.Write([]byte("errored while getting images metadata"))
			return
		}
		go func() {
			for err := range errors {
				log.Errorf("Error in the GetAllMetadata: %v", err)
			}
		}()

		// Add starting items and filter them -> we use an unbuffered channel because we don't know how many there are
		startingItems := make(chan pipe.Item)
		pipelineErrors := make(chan error)
		items := pipeline.Filter(r.Context(), startingItems, pipelineErrors)

		// adding the starting items can be done concurrently too
		started := time.Now()
		for img := range images {
			startingItems <- img
		}
		close(startingItems)

		go func() {
			for err := range pipelineErrors {
				log.Errorf("Error in the MakeGetImagePipeline: %v", err)
			}
		}()

		w.Header().Set("Content-Type", "application/json")
		counter := 0
		w.Write([]byte("{\"images\": ["))
		for item := range items {
			log.Debugf("Sending image no: %d", counter)
			counter++
			img := item.(*workers.ImageBase64)
			err := json.NewEncoder(w).Encode(img)
			if err != nil {
				w.WriteHeader(500)
			}
			w.Write([]byte(","))
		}

		w.Write([]byte("\"void\"]}"))
		w.WriteHeader(200)
		pipeline.FilteringNumber++
		pipeline.FilteringDuration += time.Since(started)
		close(pipelineErrors)
	}
}

func getImage(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {

	imagesService := workers.NewImageService(db)
	pipeline := workers.MakeGetImagePipeline(imagesService)

	return func(w http.ResponseWriter, r *http.Request) {

		imageId, err := strconv.Atoi(chi.URLParam(r, "imageId"))
		if err != nil {
			w.WriteHeader(404)
			w.Write([]byte("image id not an integer"))
			return
		}

		ch := make(chan pipe.Item, 1)
		ch <- imageId
		close(ch)
		errors := make(chan error, 1)
		started := time.Now()
		items := pipeline.Filter(r.Context(), ch, errors)
		go func() {
			for err := range errors {
				log.Errorf("Error in the MakeGetImagePipeline: %v", err)
			}
		}()

		for item := range items {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(item.(*workers.ImageBase64)); err != nil {
				w.WriteHeader(500)
			}
		}
		pipeline.FilteringNumber++
		pipeline.FilteringDuration += time.Since(started)
		close(errors)
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
