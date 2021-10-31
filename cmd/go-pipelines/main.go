package main

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ele7ija/go-pipelines/image"
	"github.com/ele7ija/go-pipelines/policy"
	"github.com/ele7ija/go-pipelines/user"
	"github.com/ele7ija/go-pipelines/user/jwt"
	pipe "github.com/ele7ija/pipeline"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	metrics "github.com/tevjef/go-runtime-metrics"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	DbHost         = "localhost"
	DbPort         = 5432
	DbUser         = "go-pipelines"
	DbPassword     = "go-pipelines"
	DbName         = "go-pipelines"
	MaxOpenConns   = 40
	InfluxHost     = "localhost"
	InfluxPort     = "8086"
	InfluxName     = "stats"
	InfluxUsername = "go-pipelines"
	InfluxPassword = "go-pipelines"
	HttpPort       = "3333"
	UserRegoPath   = filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "ele7ija", "go-pipelines", "user", "rego")
	LoadRegoPath   = filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "ele7ija", "go-pipelines", "policy", "rego")
)

func main() {

	readEnvironment()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.999",
	})
	log.SetLevel(log.DebugLevel)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.SetHeader("Access-Control-Allow-Origin", "*"))

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		DbHost, DbPort, DbUser, DbPassword, DbName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(MaxOpenConns)
	defer db.Close()

	err = pingDb(db)
	if err != nil {
		panic(err)
	}
	log.Info("Successfully connected to DB!")
	imageRequestsEngine := policy.NewImageRequestsEngine(LoadRegoPath)

	r.Mount("/api/images", imagesRouter(db, imageRequestsEngine))
	r.Mount("/api/login", userRouter(db))

	fs := http.FileServer(http.Dir("static"))
	r.Handle("/*", http.StripPrefix("", fs))

	// Collect performance stats
	go func() {
		conf := &metrics.Config{
			Host:               fmt.Sprintf("%s:%s", InfluxHost, InfluxPort),
			Database:           InfluxName,
			Username:           InfluxUsername,
			Password:           InfluxPassword,
			CollectionInterval: time.Second,
			BatchInterval:      time.Second * 15,
		}
		runCollector(conf)
	}()

	if err := http.ListenAndServe(fmt.Sprintf(":%s", HttpPort), r); err != nil {
		return
	}
}

func imagesRouter(db *sql.DB, engine policy.ImageRequestsEngine) http.Handler {

	r := chi.NewRouter()
	r.Use(UserOnly(db), AdminOnly(db), ParseForm, CheckImagePolicy(engine))
	r.Get("/", getAllImages(db))
	r.Get("/{imageId}", getImage(db))
	r.Post("/", createImages(db))
	return r
}

func userRouter(db *sql.DB) http.Handler {

	r := chi.NewRouter()
	r.Post("/", login(db))
	return r
}

// UserOnly does authentication. It puts userId and username into context.
func UserOnly(db *sql.DB) func(next http.Handler) http.Handler {
	service := user.NewService(db, UserRegoPath)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			c, err := r.Cookie("jwt")
			if err != nil {
				w.WriteHeader(403)
				log.Errorf("no jwt")

				return
			}

			u, err := service.GetUser(r.Context(), jwt.JWT(c.Value))
			if err != nil {
				w.WriteHeader(403)
				log.Errorf("bad jwt: %s", err)
				return
			}

			ctx := context.WithValue(r.Context(), "userId", u.ID)
			ctx = context.WithValue(ctx, "username", u.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminOnly allows the request only if the user is an admin
func AdminOnly(db *sql.DB) func(next http.Handler) http.Handler {
	service := user.NewService(db, UserRegoPath)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			username := r.Context().Value("username").(string)
			if username == "" {
				w.WriteHeader(403)
				w.Write([]byte("couldn't find a role"))
				log.Errorf("couldn't find a role")
				return
			}

			isAdmin, err := service.IsAdmin(r.Context(), username)
			if err != nil || !isAdmin {
				w.WriteHeader(403)
				w.Write([]byte("not allowed"))
				log.Warnf("user %s is not allowed to operate", username)
				return
			}
			log.Infof("user %s is allowed to operate", username)
			next.ServeHTTP(w, r)
		})
	}
}

// ParseForm does the form parsing for POST requests
func ParseForm(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		err := r.ParseMultipartForm(1 << 30) // 1GB
		if err != nil {
			log.Errorf("form error: %s", err)
			w.WriteHeader(400)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CheckImagePolicy checks whether image-related request is okay
// One example check is whether the maximum number of images is violated
func CheckImagePolicy(engine policy.ImageRequestsEngine) func(next http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			imageRequest := policy.ImageRequest{
				Path:   r.URL.Path,
				Method: r.Method,
				Header: r.Header,
			}
			if r.Method == http.MethodPost {
				imageRequest.SizeOfImages = r.ContentLength
			}
			if r.MultipartForm != nil {
				fhs := r.MultipartForm.File["images"]
				imageRequest.NumberOfImages = len(fhs)
			}
			b, err := engine.IsAllowed(r.Context(), imageRequest)
			if b != true || err != nil {
				w.WriteHeader(403)
				w.Write([]byte("not allowed"))
				if err != nil {
					log.Errorf("policy error happened: %s", err)
				}
				if b != true {
					log.Warnf("policy decided false")
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func login(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {

	service := user.NewService(db, UserRegoPath)

	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("jwt")
		if err == nil {
			u, err := service.GetUser(r.Context(), jwt.JWT(cookie.Value))
			if err == nil {
				w.WriteHeader(201)
				log.Infof("user '%s' already logged in.", u.Username)
				return
			}
		}

		r.ParseMultipartForm(1 << 10)
		username := r.Form.Get("username")
		password := r.Form.Get("password")

		if username == "" || password == "" {
			log.Errorf("empty username or password")
			w.WriteHeader(400)
			return
		}

		password, err = encodeS256(password)
		if err != nil {
			log.Errorf("couldn't hash password")
			w.WriteHeader(500)
			return
		}
		createdJwt, err := service.Login(r.Context(), user.User{
			Username: username,
			Password: password,
		})
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("login error"))
			log.Errorf("login error: %s", err)
			return
		}
		w.Header().Set("Set-Cookie", fmt.Sprintf("%s=%s; Max-Age=%d", "jwt", createdJwt, 60*60*2))
		w.WriteHeader(200)
		w.Write([]byte("succ login"))
		log.Infof("Logged in user: %s", username)
	}
}

func createImages(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {

	imagesService := image.NewImageService(db)
	pipeline := image.MakeCreateImagesPipelineBoundedFilters(imagesService)

	return createImagesWithPipeline(pipeline)
}

func createImagesWithPipeline(pipeline *pipe.Pipeline) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		// asynchronously add starting items
		fhs := r.MultipartForm.File["images"]
		startingItems := make(chan pipe.Item, len(fhs))
		go func() {
			for _, fh := range fhs {
				startingItems <- fh
			}
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
			img := item.(*image.ImageBase64)
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

	imagesService := image.NewImageService(db)
	pipeline := image.MakeGetAllImagesPipeline(imagesService)

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
			img := item.(*image.ImageBase64)
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

	imagesService := image.NewImageService(db)
	pipeline := image.MakeGetImagePipeline(imagesService)

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
			if err := json.NewEncoder(w).Encode(item.(*image.ImageBase64)); err != nil {
				w.WriteHeader(500)
			}
		}
		pipeline.FilteringNumber++
		pipeline.FilteringDuration += time.Since(started)
		close(errors)
	}
}

func encodeS256(password string) (string, error) {
	h := sha1.New()
	_, err := h.Write([]byte(password))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readEnvironment() {
	if envDbHost := os.Getenv("DB_HOST"); envDbHost != "" {
		DbHost = envDbHost
	}
	if envDbPort := os.Getenv("DB_PORT"); envDbPort != "" {
		DbPort, _ = strconv.Atoi(envDbPort)
	}
	if envDbUsername := os.Getenv("DB_USERNAME"); envDbUsername != "" {
		DbUser = envDbUsername
	}
	if envDbPassword := os.Getenv("DB_PASSWORD"); envDbPassword != "" {
		DbPassword = envDbPassword
	}
	if envDbName := os.Getenv("DB_NAME"); envDbName != "" {
		DbName = envDbName
	}
	if envHttpPort := os.Getenv("HTTP_PORT"); envHttpPort != "" {
		HttpPort = envHttpPort
	}
	if envInHost := os.Getenv("INFLUX_HOST"); envInHost != "" {
		InfluxHost = envInHost
	}
	if envInPort := os.Getenv("INFLUX_PORT"); envInPort != "" {
		InfluxPort = envInPort
	}
	if envInName := os.Getenv("INFLUX_NAME"); envInName != "" {
		InfluxName = envInName
	}
	if envInUsername := os.Getenv("INFLUX_USERNAME"); envInUsername != "" {
		InfluxUsername = envInUsername
	}
	if envInPassword := os.Getenv("INFLUX_PASSWORD"); envInPassword != "" {
		InfluxPassword = envInPassword
	}
	if envUserRego := os.Getenv("USER_REGO_PATH"); envUserRego != "" {
		UserRegoPath = envUserRego
	}
	if envLoadRego := os.Getenv("LOAD_REGO_PATH"); envLoadRego != "" {
		LoadRegoPath = envLoadRego
	}
}

func runCollector(conf *metrics.Config) {
	for i := 0; i < 10; i++ {
		if err := metrics.RunCollector(conf); err != nil {
			log.Errorf("An error happened while sending performance stats to InfluxDB: %s", err)
		} else {
			log.Infof("Successfully connected to InfluxDB, sending data...")
			break
		}
		<-time.After(time.Second * 5)
		log.Warnf("Waited 5s, pinging Influx again...")
	}
}

func pingDb(db *sql.DB) (err error) {
	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			log.Infof("Successfully connected to DB, sending data...")
			return
		}
		<-time.After(time.Second * 3)
		log.Warnf("Waited 3s, pinging DB again...")
	}
	return
}
