package main

import (
	"log"
	"os"
	loaddatabase "stock-agent/load-database"

	"github.com/joho/godotenv"
)

func main() {

	// pull in the env vars
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("error loading .env file:", err)
	}

	// load the database
	load, exists := os.LookupEnv("LOAD_DB")
	if exists && load == "true" {
		loaddatabase.LoadNasdaqDatabase("nasdaq")
	}
}
