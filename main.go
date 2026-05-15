package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Read   bool   `json:"read"`
}

var books = map[string]Book{
	"1": Book{ID: "1", Title: "The Way of Kings", Author: "Brandon Sanderson", Read: true},
	"4": Book{ID: "4", Title: "Lord of the Rings", Author: "J.R.Tolkien", Read: false},
}

// endpoints to implement:
// GET 		/books: 		Return all books as a JSON array
func getBookHandler(w http.ResponseWriter, r *http.Request) {
	listedBooks := make([]Book, 0)

	for _, book := range books {
		listedBooks = append(listedBooks, book)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(listedBooks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// POST 	/books: 		Add a new book (body contains title+author)
// GET 		/books/{id}: 	Return one book by ID
// PUT		/books/{id}:	Mark a book as read
// DELETE	/books/{id}:	Remove a book

func main() {
	r := chi.NewRouter()

	r.Get("/books", getBookHandler)

	fmt.Println("Listening...")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Println(err)
	}
}
