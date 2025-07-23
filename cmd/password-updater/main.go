package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "user=postgres password=postgres host=localhost port=5433 dbname=bookwork sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("UPDATE users SET password_hash = $1 WHERE email = $2", "$2a$10$2Yrl7Of7T1Zk/zfi0ZhWeO1hkq92fhoEdrsyrSmvH1VfqoHfLPaCu", "admin@bookwork.com")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Password updated successfully")
}
