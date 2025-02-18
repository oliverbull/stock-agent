package main

import (
	"context"
	"log"
	"os"
	datacombineagent "stock-agent/data-combine-agent"
	databaseagent "stock-agent/database-agent"
	loaddatabase "stock-agent/load-database"
	quarterlyresultsagent "stock-agent/quarterly-results-agent"

	"github.com/joho/godotenv"
)

func main() {

	// pull in the env vars
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("error loading .env file:", err)
	}

	// load the database if requested
	load, exists := os.LookupEnv("LOAD_DB")
	if exists && load == "true" {
		loaddatabase.LoadNasdaqDatabase("nasdaq")
	}

	// agent initializations
	// initialize and run the nasdaq database agent as a service
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

	// initialize and run the quarterly results agent as a service
	qrAgentHostname, exists := os.LookupEnv("QUARTERLY_RESULTS_AGENT_HOSTNAME")
	if !exists {
		log.Fatalln("missing QUARTERLY_RESULTS_AGENT_HOSTNAME in .env")
	}
	qrAgentPort, exists := os.LookupEnv("QUARTERLY_RESULTS_AGENT_PORT")
	if !exists {
		log.Fatalln("missing QUARTERLY_RESULTS_AGENT_PORT in .env")
	}
	qrAgent, err := quarterlyresultsagent.InitQuarterlyResultsAgent(context.Background())
	if err != nil {
		log.Fatalln("error InitQuarterlyResultsAgent:", err)
	}
	qrAgent.RunAgent(qrAgentHostname, qrAgentPort)

	// initialize and run the data combiner agent as a service
	dcAgentHostname, exists := os.LookupEnv("DATA_COMBINE_AGENT_HOSTNAME")
	if !exists {
		log.Fatalln("missing DATA_COMBINE_AGENT_HOSTNAME in .env")
	}
	dcAgentPort, exists := os.LookupEnv("DATA_COMBINE_AGENT_PORT")
	if !exists {
		log.Fatalln("missing DATA_COMBINE_AGENT_PORT in .env")
	}
	dcAgent, err := datacombineagent.InitDataCombineAgent(context.Background())
	if err != nil {
		log.Fatalln("error InitDatInitDataCombineAgentabaseAgent:", err)
	}
	dcAgent.RunAgent(dcAgentHostname, dcAgentPort)

	// call the database agent through the client tool
	//response, err := databaseagent.CallDatabaseAgent("what was Apple's highest close price in November 2024")
	//response, err := databaseagent.CallDatabaseAgent("how many collections are there")
	if err != nil {
		log.Fatalln("error call agent:", err)
	}
	//log.Println(response)

	// call the quarterly results agent through the client tool
	//response, err = quarterlyresultsagent.CallQuarterlyResultsAgent("Get Apple's Q4 2024 results")
	if err != nil {
		log.Fatalln("error call agent:", err)
	}
	//log.Println(response)

	// call the data combiner for a compound query
	response, err := datacombineagent.CallDataCombineAgent("Get Apple's close price for all of November 2024, summarize the same years Q4 results and then generate a table for all quarters of 2024 financial results")
	if err != nil {
		log.Fatalln("error call agent:", err)
	}
	log.Println(response)
}
