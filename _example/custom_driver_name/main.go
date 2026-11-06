package main

import (
	"database/sql"

	_ "github.com/CovenantSQL/go-sqlite3-encrypt"
)

func main() {
	for _, driver := range sql.Drivers() {
		println(driver)
	}
}
