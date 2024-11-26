package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {

	//create the database
	if err := createDB(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func loadFile(tableName string, fileName string, dbName string) {
	fmt.Printf("Loading %s table with %s\n", tableName, fileName)

	//open csv file
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	//prepare db for inserting data
	var insertSQL string

	switch {
	case tableName == "actors":
		insertSQL = `INSERT INTO actors (id, first_name, last_name, gender) VALUES (?, ?, ?, ?)`
	case tableName == "movies":
		insertSQL = `INSERT INTO movies (id, name, year, rank) VALUES (?, ?, ?, ?)`
	case tableName == "directors":
		insertSQL = `INSERT INTO directors (id, first_name, last_name) VALUES (?, ?, ?)`
	case tableName == "roles":
		insertSQL = `INSERT INTO roles (actor_id, movie_id, role) VALUES (?, ?, ?)`
	case tableName == "movies_genres":
		insertSQL = `INSERT INTO movies_genres (movie_id, genre) VALUES (?, ?)`
	case tableName == "directors_genres":
		insertSQL = `INSERT INTO directors_genres (director_id, genre, prob) VALUES (?, ?, ?)`
	}

	statement, err := db.Prepare(insertSQL)
	if err != nil {
		fmt.Println("Error preparing SQL statement:", err)
		return
	}
	defer statement.Close()

	//loop through records and insert into db
	// Skip header row
	records = records[1:]
	for i, record := range records {
		values := make([]interface{}, len(record))
		for i, v := range record {
			values[i] = v
		}
		_, err := statement.Exec(values...)
		if err != nil {
			fmt.Println("Error inserting record:", err, values)
			return
		}
		if i%1000 == 0 {
			fmt.Printf("Inserted %d records(%.2f%%)\n", i, float64(i)/float64(len(records))*100)
		}
	}
}

// bulk of main functionality
func createDB() error {

	//.db name
	fn := "./movies.db"

	//check if db exists. If so, return nil
	_, err := os.Stat(fn)
	if err == nil {
		fmt.Println("Database already exists. Skipping Creation Process")
		return nil
	}

	// check if db exists and open. If not, create the file
	db, err := sql.Open("sqlite", fn)
	if err != nil {
		return err
	}

	//create tables
	createTablesSQL := []string{
		`CREATE TABLE IF NOT EXISTS actors (
            "id" INTEGER NOT NULL PRIMARY KEY,
            "first_name" TEXT,
			"last_name" TEXT,
			"gender" TEXT
        );`,
		`CREATE TABLE IF NOT EXISTS roles (
            "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
            "actor_id" INTEGER,
            "movie_id" INTEGER,
            "role" TEXT,
            FOREIGN KEY(actor_id) REFERENCES actors(id),
            FOREIGN KEY(movie_id) REFERENCES movies(id)
        );`,
		`CREATE TABLE IF NOT EXISTS movies (
            "id" INTEGER NOT NULL PRIMARY KEY,
            "name" TEXT,
            "year" INTEGER,
			"rank" INTEGER
        );`,
		`CREATE TABLE IF NOT EXISTS movies_genres (
            "movie_id" INTEGER,
            "genre" TEXT,
			PRIMARY KEY(movie_id, genre),
            FOREIGN KEY(movie_id) REFERENCES movies(id)
        );`,
		`CREATE TABLE IF NOT EXISTS directors (
            "id" INTEGER NOT NULL PRIMARY KEY,
            "first_name" TEXT,
			"last_name" TEXT
        );`,
		`CREATE TABLE IF NOT EXISTS directors_genres (
            "director_id" INTEGER,
            "genre" TEXT,
			"prob" FLOAT,
            FOREIGN KEY(director_id) REFERENCES directors(id)
        );`,
	}

	//execute create table statements
	for _, sqlStmt := range createTablesSQL {
		_, err := db.Exec(sqlStmt)
		if err != nil {
			return err
		}
	}

	fmt.Println("Tables Successfully created")

	//close db
	db.Close()

	//populate tables using csv files
	loadFile("actors", "./data/IMDB-actors.csv", fn)
	loadFile("movies", "./data/IMDB-movies.csv", fn)
	loadFile("directors", "./data/IMDB-directors.csv", fn)
	loadFile("roles", "./data/IMDB-roles.csv", fn)
	loadFile("movies_genres", "./data/IMDB-movies_genres.csv", fn)
	loadFile("directors_genres", "./data/IMDB-directors_genres.csv", fn)

	fmt.Println("Tables Successfully loaded")

	return nil
}
