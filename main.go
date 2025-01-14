package main

import (
	"context"
	"log"
	"os"
	databaseagent "stock-agent/database-agent"
	loaddatabase "stock-agent/load-database"
	"time"

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

	// initialize and run the nasdaq database agent
	dbAgentHostname, exists := os.LookupEnv("DATABASE_AGENT_HOSTNAME")
	if !exists {
		log.Fatalln("missing DATABASE_AGENT_HOSTNAME in .env")
	}
	dbAgentPort, exists := os.LookupEnv("DATABASE_AGENT_PORT")
	if !exists {
		log.Fatalln("missing DATABASE_AGENT_PORT in .env")
	}
	dbAgent, err := databaseagent.InitDatabaseAgent(context.Background())
	if err != nil {
		log.Fatalln("error InitDatabaseAgent:", err)
	}
	dbAgent.RunAgent(dbAgentHostname, dbAgentPort)

	time.Sleep(1 * time.Second)

	// call the database agent through the client tool
	response, err := databaseagent.CallDatabaseAgent("what was Apple's highest close price in November 2024")
	//response, err := databaseagent.CallDatabaseAgent("how many collections are there")
	if err != nil {
		log.Fatalln("error CallAgent:", err)
	}
	log.Println(response)
}
