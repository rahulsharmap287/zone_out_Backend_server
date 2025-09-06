package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Product struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type Order struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Items     []Product `json:"items"`
	CreatedAt time.Time `json:"created_at"`
	Hidden    bool      `json:"hidden"`
}

var (
	orders      = []Order{}
	ordersMu    sync.Mutex
	nextOrderID = 1
)

// ✅ CORS
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// ✅ Serve Images
func serveImagesFromFolder(w http.ResponseWriter, folder, route string) {
	files, err := os.ReadDir(folder)
	if err != nil {
		http.Error(w, "Failed to read images directory", http.StatusInternalServerError)
		return
	}

	var products []Product
	id := 1
	for _, file := range files {
		if !file.IsDir() {
			url := "http://192.168.1.12:8080/images/" + route + "/" + file.Name()
			products = append(products, Product{ID: id, URL: url})
			id++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// ✅ Hide Order (Admin Only)
func hideOrderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	ordersMu.Lock()
	defer ordersMu.Unlock()

	for i := range orders {
		if orders[i].ID == id {
			orders[i].Hidden = true
			break
		}
	}
	w.WriteHeader(http.StatusOK)
}

// ✅ Orders (User + Admin)
func ordersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		username := r.URL.Query().Get("username")

		ordersMu.Lock()
		defer ordersMu.Unlock()

		var result []Order
		for _, o := range orders {
			if username == "admin" {
				result = append(result, o)
			} else if o.Username == username && !o.Hidden {
				result = append(result, o)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)

	case http.MethodPost:
		var in Order
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(in.Username) == "" {
			http.Error(w, "username required", http.StatusBadRequest)
			return
		}

		ordersMu.Lock()
		in.ID = nextOrderID
		nextOrderID++
		in.CreatedAt = time.Now()
		orders = append(orders, in)
		ordersMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(in)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ✅ Delete Order by ID
func orderByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/orders/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ordersMu.Lock()
	defer ordersMu.Unlock()

	idx := -1
	for i, o := range orders {
		if o.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	orders = append(orders[:idx], orders[idx+1:]...)
	w.WriteHeader(http.StatusNoContent)
}

// ✅ Delete all orders by username (Admin bulk delete)
func deleteUserSelections(w http.ResponseWriter, r *http.Request) {
	uname := r.URL.Query().Get("username")
	if uname == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	ordersMu.Lock()
	defer ordersMu.Unlock()

	newOrders := []Order{}
	for _, o := range orders {
		if o.Username != uname {
			newOrders = append(newOrders, o)
		}
	}
	orders = newOrders

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("All orders for user deleted"))
}

func main() {
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))

	http.HandleFunc("/api/keychains", func(w http.ResponseWriter, r *http.Request) {
		serveImagesFromFolder(w, "./images/Keychains", "Keychains")
	})
	http.HandleFunc("/api/stickers", func(w http.ResponseWriter, r *http.Request) {
		serveImagesFromFolder(w, "./images/Stickers", "Stickers")
	})
	http.HandleFunc("/api/PocketWatch", func(w http.ResponseWriter, r *http.Request) {
		serveImagesFromFolder(w, "./images/PocketWatch", "PocketWatch")
	})
	http.HandleFunc("/api/Bracelet", func(w http.ResponseWriter, r *http.Request) {
		serveImagesFromFolder(w, "./images/Bracelet", "Bracelet")
	})
	http.HandleFunc("/api/Lockets", func(w http.ResponseWriter, r *http.Request) {
		serveImagesFromFolder(w, "./images/Lockets", "Lockets")
	})
	http.HandleFunc("/api/Posters", func(w http.ResponseWriter, r *http.Request) {
		serveImagesFromFolder(w, "./images/Posters", "Posters")
	})
	http.HandleFunc("/api/Anime", func(w http.ResponseWriter, r *http.Request) {
		serveImagesFromFolder(w, "./images/Anime", "Anime")
	})
	http.HandleFunc("/api/Polaroids", func(w http.ResponseWriter, r *http.Request) {
		serveImagesFromFolder(w, "./images/Polaroids", "Polaroids")
	})
	http.HandleFunc("/api/Albums", func(w http.ResponseWriter, r *http.Request) {
		serveImagesFromFolder(w, "./images/Albums", "Albums")
	})

	http.HandleFunc("/api/orders", ordersHandler)
	http.HandleFunc("/api/orders/", orderByIDHandler)
	http.HandleFunc("/api/hideOrder", hideOrderHandler)
	http.HandleFunc("/api/deleteUserSelections", deleteUserSelections)

	addr := "192.168.1.12:8080"
	log.Println("🚀 Server running at http://" + addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(http.DefaultServeMux)))
}
