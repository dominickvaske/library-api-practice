package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

type App struct {
	db *sql.DB
}

type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Read   bool   `json:"read"`
}

// GET /books: Return all books as a JSON array
func (a *App) getBookHandler(w http.ResponseWriter, r *http.Request) {
	listedBooks := make([]Book, 0)

	rows, err := a.db.Query(`select id, title, author, read from books`)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		b := Book{}
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Read); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		listedBooks = append(listedBooks, b)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(listedBooks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// POST /books: Add a new book (body contains title+author)
func (a *App) newBookHandler(w http.ResponseWriter, r *http.Request) {
	var b Book
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	b.Read = false

	query := `insert into books (title, author, read) values ($1, $2, $3) returning id`
	row := a.db.QueryRow(query, b.Title, b.Author, b.Read)
	if err := row.Scan(&b.ID); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GET /books/{id}: Return one book by ID
func (a *App) getSpecificBook(w http.ResponseWriter, r *http.Request) {
	idInt, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var b Book
	query := `select id, title, author, read from books where id = $1`
	err = a.db.QueryRow(query, idInt).Scan(&b.ID, &b.Title, &b.Author, &b.Read)

	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "book not found", http.StatusNotFound)
	} else if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(b); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// PUT /books/{id}: Mark a book as read
func (a *App) markRead(w http.ResponseWriter, r *http.Request) {
	idInt, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	result, err := a.db.Exec(`update books set read=$1 where id=$2`, true, idInt)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		http.Error(w, "book not found", http.StatusNotFound)
		return
	}

	var book Book
	err = a.db.QueryRow(`select id, title, author, read from books where id=$1`, idInt).
		Scan(&book.ID, &book.Title, &book.Author, &book.Read)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(book); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// DELETE /books/{id}: Remove a book
func (a *App) deleteBook(w http.ResponseWriter, r *http.Request) {
	idInt, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var book Book
	err = a.db.QueryRow(`select id, title, author, read from books where id=$1`, idInt).
		Scan(&book.ID, &book.Title, &book.Author, &book.Read)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "book not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if _, err = a.db.Exec(`delete from books where id=$1`, idInt); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(book); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	app := App{db: db}

	r := chi.NewRouter()
	r.Get("/books", app.getBookHandler)
	r.Post("/books", app.newBookHandler)
	r.Get("/books/{id}", app.getSpecificBook)
	r.Put("/books/{id}", app.markRead)
	r.Delete("/books/{id}", app.deleteBook)

	fmt.Println("Listening...")
	if err = http.ListenAndServe(":8080", r); err != nil {
		fmt.Println(err)
	}
}