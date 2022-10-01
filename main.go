package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type ShortenRequest struct {
	Url string `json:"url"`
}

type ShortenResponse struct {
	Url string `json:"url"`
}

func main() {
	router := chi.NewRouter()
	
	router.Use(middleware.Logger)
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)

	router.Post("/shorten", func (w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)

		if err != nil {
			http.Error(w, "There was an error shortening the URL.", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}

		request := &ShortenRequest{}

		if jsonErr := json.Unmarshal(body, request); jsonErr != nil {
			http.Error(w, jsonErr.Error(), http.StatusInternalServerError)
			fmt.Println(jsonErr)
			return
		}

		if request.Url == "" {
			http.Error(w, "You need to supply the `url` field in the request body!", http.StatusBadRequest)
		}

		guid := uuid.New().String()[0:5]

		w.Header().Add("Access-Control-Allow-Origin", "*")

		response := &ShortenResponse{
			Url: fmt.Sprintf("http://%s/%s", r.Host, guid),
		}

		responseJson, err := json.Marshal(response)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err)
			return
		} 

		w.Write(responseJson)

		router.Get(fmt.Sprintf("/%s", guid), func (w http.ResponseWriter, r *http.Request) {
			HandleShortUrlRequest(w, r, request)		
		})		

		router.Post(fmt.Sprintf("/%s", guid), func(w http.ResponseWriter, r *http.Request) {
			HandleShortUrlRequest(w, r, request)
		})
	})

	http.ListenAndServe(":3000", router)
}

func HandleShortUrlRequest(w http.ResponseWriter, r *http.Request, request *ShortenRequest) {
	proxy, err := ReverseProxy(request.Url)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	proxy.ServeHTTP(w, r)
}

func ReverseProxy(proxyUrl string) (*httputil.ReverseProxy, error) {
	address, err := url.Parse(proxyUrl)

	if err != nil {
		return &httputil.ReverseProxy{}, errors.New("There was an error fetching the shortened URL.")		
	}

	p := httputil.NewSingleHostReverseProxy(address)

	p.Director = func(request *http.Request) {
		request.Host = address.Host
		request.URL.Scheme = address.Scheme
		request.URL.Host = address.Host
		request.URL.Path = address.Path
	}

	return p, nil
}
