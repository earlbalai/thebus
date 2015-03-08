/*
* The Bus Information Agent
* Created by Earl Balai Jr
 */
package main

import (
	"database/sql"
	"fmt"

	_ "github.com/bmizerany/pq"
)

const (
	DB_NAME     = "table_name"
	DB_USER     = "user"
	DB_PASSWORD = "password"
)

func OpenDB() *sql.DB {
	db, err := sql.Open("postgres", fmt.Sprintf("dbname=%s user=%s password=%s sslmode=disable", DB_NAME, DB_USER, DB_PASSWORD))
	if err != nil {
		panic(err)
	}
	return db
}

func DBTest() {
	fmt.Printf("Testing connection to database with DB: %s as USER: %s \n", DB_NAME, DB_USER)
	db := OpenDB()
	fmt.Println("Connection successful! \n")

	defer db.Close()
}
