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

var books = map[string]Book{
	"1": Book{ID: 1, Title: "The Way of Kings", Author: "Brandon Sanderson", Read: true},
	"2": Book{ID: 2, Title: "Lord of the Rings", Author: "J.R.Tolkien", Read: false},
}

// endpoints to implement:
// GET 		/books: 		Return all books as a JSON array
func (a *App) getBookHandler(w http.ResponseWriter, r *http.Request) {
	listedBooks := make([]Book, 0)

	// query table and grab set of results
	q := `select id, title, author, read from books;`
	rows, err := a.db.Query(q)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer rows.Close()

	// scan next row
	for rows.Next() {
		// save row into book
		b := Book{}
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Read); err != nil {
			log.Fatal(err)
		}
		listedBooks = append(listedBooks, b)
	}

	// check error
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	// encode and send results back
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(listedBooks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// POST 	/books: 		Add a new book (body contains title+author)
func (a *App) newBookHandler(w http.ResponseWriter, r *http.Request) {
	// decode posting from request into a new book
	var b Book
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
	}
	b.Read = false

	// insert book into table, using variable insertion of query and then scan to get
	// the generated ID -- prevent SQL injection using placeholders
	query := `insert into books (title, author, read) values ($1, $2, $3) returning id`
	row := a.db.QueryRow(query, b.Title, b.Author, b.Read)
	if err := row.Scan(&b.ID); err != nil {
		log.Fatal(err)
	}

	// send a confirmation to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GET 		/books/{id}: 	Return one book by ID
func (a *App) getSpecificBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		log.Fatal(err)
	}

	//if book, ok := books[id]; !ok {
	//	http.Error(w, "book not found", http.StatusNotFound)
	//} else {
	//	w.Header().Set("Content-Type", "application/json")
	//	if err := json.NewEncoder(w).Encode(book); err != nil {
	//		http.Error(w, err.Error(), http.StatusInternalServerError)
	//	}
	//}

	var b Book
	query := `select id, title, author, read from books where id = $1`
	row := a.db.QueryRow(query, idInt)
	err = row.Scan(&b.ID, &b.Title, &b.Author, &b.Read)

	if errors.Is(err, sql.ErrNoRows) {
		// 404 not found
		http.Error(w, "book not found", http.StatusNotFound)
	} else if err != nil {
		// internal server error
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		// send content
		w.Header().Set("Content-Type", "application/json")
		err2 := json.NewEncoder(w).Encode(b)
		if err2 != nil {
			log.Fatal(err2)
		}
	}

}

// PUT		/books/{id}:	Mark a book as read
func (a *App) markRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		log.Fatal(err)
	}
	// update book
	query := `update books set read=$1 where id=$2`
	result, err := a.db.Exec(query, true, idInt)
	if err != nil {
		log.Fatal(err)
	}

	// check if no rows updated
	if rows, _ := result.RowsAffected(); rows == 0 {
		http.Error(w, "book not found", http.StatusNotFound)
	}

	var book Book
	query = `select id, title, author, read from books where id=$1`
	row := a.db.QueryRow(query, idInt)
	row.Scan(&book.ID, &book.Title, &book.Author, &book.Read)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(book); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// DELETE	/books/{id}:	Remove a book
func (a *App) deleteBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	idInt, err := strconv.Atoi(id)

	// grab the book to be deleted
	var book Book
	query := `select id, title, author, read from books where id=$1`
	row := a.db.QueryRow(query, idInt)
	err = row.Scan(&book.ID, &book.Title, &book.Author, &book.Read)
	if err != nil {
		http.Error(w, "book not found", http.StatusNotFound)
		return
	}

	// delete from database
	// no need to check if not deleted since above check
	query = `delete from books where id=$1`
	_, err = a.db.Exec(query, idInt)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(book); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// get database url and open database
	connectString := os.Getenv("DATABASE_URL")
	db, err := sql.Open("pgx", connectString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// ping to test connection
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	// create database struct
	app := App{db: db}

	// create a new router and add handlers
	r := chi.NewRouter()

	r.Get("/books", app.getBookHandler)
	r.Post("/books", app.newBookHandler)
	r.Get("/books/{id}", app.getSpecificBook)
	r.Put("/books/{id}", app.markRead)
	r.Delete("/books/{id}", app.deleteBook)

	fmt.Println("Listening...")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Println(err)
	}
}
