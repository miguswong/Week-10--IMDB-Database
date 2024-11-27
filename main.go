package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {

	//Create the database. Will skip creation process if movies.db already exists.
	if err := createDB(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// KEEP COMMENTED OUT. ONLY RUN IF NECESSARY
	// updateNullVals()

	queryTest()

}

func updateNullVals() {
	fmt.Println("Updating null values in the database")

	db, err := sql.Open("sqlite", "movies.db")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	// Update movies table to convert 'NULL' string to actual NULL value
	_, err = db.Exec("UPDATE movies SET rank = CASE WHEN rank = 'NULL' THEN NULL ELSE rank END")
	if err != nil {
		fmt.Println("Error updating null values:", err)
		os.Exit(1)
	}
}

func queryTest() error {
	fmt.Println("Querying the database")

	db, err := sql.Open("sqlite", "movies.db")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()
	// Query the database
	rows, err := db.Query(`
    WITH RankedMovies AS (
        SELECT
            m.name,
            m.rank,
            mg.genre,
            m.year,
            ROW_NUMBER() OVER (
                PARTITION BY mg.genre 
                ORDER BY m.rank DESC
            ) as rank_position
        FROM movies m
        INNER JOIN movies_genres mg ON m.id = mg.movie_id
        WHERE m.rank IS NOT NULL
    )
    SELECT
        genre,
        name,
        year,
        rank
    FROM RankedMovies
    WHERE rank_position <= 3
    ORDER BY genre, rank DESC;
	`)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return err
	}
	defer rows.Close()

	//create file to store results
	file, err := os.Create("query_results.csv")
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return err
	}
	defer file.Close()

	// Process the results
	for rows.Next() {
		var genre, name string
		var year int
		var rank float64

		err := rows.Scan(&genre, &name, &year, &rank)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		//write results to file
		_, err = file.WriteString(fmt.Sprintf("%s,\"%s\",%d,%.1f\n", genre, name, year, rank))
		if err != nil {
			log.Printf("Error printing row: %v", err)
			continue
		}
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating over rows: %v", err)
		return err
	}

	fmt.Println("Successfully exported query results.")

	return nil
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
	reader.LazyQuotes = true    // Allow quotes within fields
	reader.FieldsPerRecord = -1 // Allow variable number of fields per record

	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Reading Error: ", err)
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
		if i%10000 == 0 {
			fmt.Printf(" %s - Inserted %d records(%.2f%%)\n", tableName, i, float64(i)/float64(len(records))*100)
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
