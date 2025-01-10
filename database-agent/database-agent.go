package databaseagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	agentassemble "stock-agent/gemini-agent-assemble"

	"github.com/google/generative-ai-go/genai"
)

/////////////////
// database agent

// query database tool description
var queryDatabaseTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{{
		Name:        "queryDatabase",
		Description: "Query the database with the supplied parameters",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"ticker": {
					Type:        genai.TypeString,
					Description: "The ticker code of the company for the query",
				},
				"startDate": {
					Type:        genai.TypeString,
					Description: "The start date for a range query in the format yyyy-mm-dd",
				},
				"endDate": {
					Type:        genai.TypeString,
					Description: "The end date for a range query in the format yyyy-mm-dd",
				},
			},
			Required: []string{"ticker", "startDate", "endDate"},
		},
	}},
}

// query database tool
func queryDatabase(ticker string, startDate string, endDate string) string {
	log.Println("running queryDatabase tool for " + ticker + " with date range " + startDate + " - " + endDate)
	// ToDo: add database query here
	return "ToDo: database query result"
}

// agent initialization
func InitDatabaseAgent(ctx context.Context) (*agentassemble.Agent, error) {
	system := `ToDo: system prompt`
	var tools = []*genai.Tool{queryDatabaseTool}
	agentDatabase, err := agentassemble.InitAgent(ctx, &system, tools, callDatabaseTool)
	if err != nil {
		log.Println("Error initializing the float agent")
		return nil, err
	}
	return agentDatabase, err
}

// tool call handler
func callDatabaseTool(funcall genai.FunctionCall) (string, error) {

	result := ""
	// find the function to call - all tool calls come here
	if funcall.Name == queryDatabaseTool.FunctionDeclarations[0].Name {
		// check the params are populated
		ticker, exists := funcall.Args["ticker"]
		if !exists {
			log.Fatalln("Missing ticker")
		}
		startDate, exists := funcall.Args["startDate"]
		if !exists {
			log.Fatalln("Missing start date")
		}
		endDate, exists := funcall.Args["endDate"]
		if !exists {
			log.Fatalln("Missing end date")
		}
		// call the calc tool
		result = queryDatabase(ticker.(string), startDate.(string), endDate.(string))
		log.Println("query database result: " + result)
	} else {
		log.Println("unhandled function name: " + funcall.Name)
		return "", errors.New("unhandled function name: " + funcall.Name)
	}
	return result, nil
}

//////////////////////////////////////////
// client tools for external agents to use

// database agent client tool description
var CallDatabaseAgentTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{{
		Name:        "callDatabaseAgent",
		Description: "Make a request to the database agent. The agent will perform the requested query and return the result.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"message": {
					Type:        genai.TypeString,
					Description: "The natural language request message for the database agent",
				},
			},
			Required: []string{"message"},
		},
	}},
}

// client tool for the database agent
func CallDatabaseAgent(message string) (string, error) {
	log.Println("running callDatabaseAgent tool for :" + message)

	// get the float agent endpoint
	floatHostname, ok := os.LookupEnv("DATABASE_AGENT_HOSTNAME")
	if !ok {
		log.Fatalln("environment variable DATABASE_AGENT_HOSTNAME not set")
	}
	floatPort, ok := os.LookupEnv("DATABASE_AGENT_PORT")
	if !ok {
		log.Fatalln("environment variable DATABASE_AGENT_PORT not set")
	}

	// build the payload
	request := agentassemble.Request{
		Input: message,
	}
	reqDat, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// repare the request
	req, err := http.NewRequest("POST", "http://"+floatHostname+":"+floatPort+"/agent", bytes.NewBuffer(reqDat))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// send the post
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// extract and decode the reply
	response := agentassemble.Response{}
	respDat, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(respDat, &response)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}
