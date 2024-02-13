package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt"
	"html/template"
	"justHTTP/internal/storage"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Server struct {
	users   map[string]string
	storage storage.Storage
	tmp     *template.Template
}

func NewServer() *Server {
	return &Server{
		storage: *storage.New(),
		users:   make(map[string]string),
		tmp:     template.Must(template.ParseGlob("internal/templates/*.html")),
	}
}

func main() {
	server := NewServer()
	//router := http.NewServeMux()
	router := mux.NewRouter()

	auth := router.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/register", server.Register).Methods(http.MethodPost)
	auth.HandleFunc("/login", server.Signin).Methods(http.MethodPost)

	itemRouter := router.NewRoute().Subrouter()
	itemRouter.Use(authorizeMiddleware)
	itemRouter.HandleFunc("/welcome", server.WelcomeHandler).Methods("GET")
	itemRouter.HandleFunc("/item", server.GetAllItemsHandler).Methods("GET")
	itemRouter.HandleFunc("/item", server.DeleteAll).Methods("DELETE")
	itemRouter.HandleFunc("/item", server.AddItemHandler).Methods("POST")

	itemRouter.HandleFunc("/item/{id}", server.GetItemByIdHandler).Methods("GET")
	itemRouter.HandleFunc("/item/{id}", server.DeleteItemById).Methods("DELETE")

	log.Fatal(http.ListenAndServe("localhost:8080", router))
}

func (s *Server) WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "No jwt token", http.StatusBadRequest)
		return
	}
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		http.Error(w, "bad token", http.StatusBadRequest)
		return
	}
	claims := parsed.Claims.(jwt.MapClaims)
	_ = s.tmp.ExecuteTemplate(w, "welcome.html", claims["sub"])
	//fmt.Fprintln(w, "Hello,", claims["sub"])
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "bad credentials", http.StatusBadRequest)
		return
	}
	if _, ok := s.users[req.Username]; ok {
		http.Error(w, "username already taken", http.StatusBadRequest)
		return
	}
	s.users[req.Username] = req.Password
	w.WriteHeader(http.StatusOK)
	return
}

func (s *Server) Signin(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "bad credentials", http.StatusBadRequest)
		return
	}
	if pass, ok := s.users[req.Username]; !ok || req.Password != pass {
		http.Error(w, "login failed", http.StatusBadRequest)
		return
	}
	payload := jwt.MapClaims{
		"sub": req.Username,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	strToken, err := token.SignedString([]byte("secret"))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	type Response struct {
		Token string `json:"JWT"`
	}
	renderJSON(w, Response{Token: strToken})
}

func authorizeMiddleware(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token, err := jwt.Parse(header, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})
		if err != nil {
			if errors.Is(err, jwt.ErrSignatureInvalid) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !token.Valid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func middleware(one http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		log.Println(r.Method)
	})
}

//func (s *Server) ItemHandler(w http.ResponseWriter, r *http.Request) {
//	if r.URL.Path == "/item/" {
//		if r.Method == http.MethodGet {
//			s.GetAllItemsHandler(w, r)
//		} else if r.Method == http.MethodPost {
//			s.AddItemHandler(w, r)
//		} else if r.Method == http.MethodDelete {
//			s.DeleteAll(w, r)
//		} else {
//			http.Error(w, "invalid operation", http.StatusBadRequest)
//		}
//	} else {
//		strs := strings.Split(r.URL.Path, "/")
//		if len(strs) < 2 {
//			http.Error(w, "invalid URL", http.StatusBadRequest)
//			return
//		}
//		id, err := strconv.Atoi(strs[2])
//		if err != nil {
//			http.Error(w, "invalid URL", http.StatusBadRequest)
//			return
//		}
//		if r.Method == http.MethodGet {
//			s.GetItemByIdHandler(w, r, id)
//		} else if r.Method == http.MethodDelete {
//			s.DeleteItemById(w, r, id)
//		}
//	}
//}

func (s *Server) GetAllItemsHandler(w http.ResponseWriter, r *http.Request) {
	items := s.storage.GetAll()
	renderJSON(w, items)
}

func (s *Server) AddItemHandler(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Info string
	}

	type Response struct {
		Id int
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, ("application/json expected got " + contentType), http.StatusUnsupportedMediaType)
		return
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req Request
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}
	id := s.storage.AddItem(req.Info)
	renderJSON(w, Response{Id: id})
}

func (s *Server) DeleteAll(w http.ResponseWriter, r *http.Request) {
	s.storage.DeleteAll()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) GetItemByIdHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	item, err := s.storage.GetItem(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("cant find item %d", id), http.StatusBadRequest)
		return
	}
	renderJSON(w, item)
}

func (s *Server) DeleteItemById(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	err := s.storage.DeleteItem(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	w.Write([]byte("OK"))
}

func renderJSON(w http.ResponseWriter, v any) {
	log.Println(v)
	js, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println(js)
	w.Header().Add("Content-Type", "application/json")
	w.Write(js)
}
