package quarterlyresultsagent

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

//////////////////////////
// quarterly results agent

// specific data range query database tool description
var quarterlyResultsTools = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{{
		Name:        "getResults",
		Description: "get the ticker's quarterly results.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"ticker": {
					Type:        genai.TypeString,
					Description: "The ticker code of the company",
				},
				"year": {
					Type:        genai.TypeString,
					Description: "The year for the results in the format yyyy",
				},
				"quarter": {
					Type:        genai.TypeString,
					Description: "The quarter number in the format q-n where n is the quarter number",
				},
			},
			Required: []string{"ticker", "year", "quarter"},
		},
	}},
}

// tool to check if a quarterly result is available
func getResults(ticker string, year string, quarter string) string {
	log.Println("running getResults tool for " + ticker + " for " + quarter + " - " + year)

	// ToDo: get the results

	return "todo"
}

// agent initialization
func InitQuarterlyResultsAgent(ctx context.Context) (*agentassemble.Agent, error) {
	system := `
You are an AI agent that retrieve a stock ticker's quarterly results.
You must use the tools to help answer the request and retrun the result.
`
	// initialize the agent
	var tools = []*genai.Tool{quarterlyResultsTools}
	agentQuarterlyResults, err := agentassemble.InitAgent(ctx, &system, tools, quarterlyResultsTool)
	if err != nil {
		log.Println("Error initializing the quarterly results agent")
		return nil, err
	}

	// always start a new session
	agentQuarterlyResults.NewSession()

	return agentQuarterlyResults, err
}

// tool call handler
func quarterlyResultsTool(funcall genai.FunctionCall) (string, error) {

	result := ""
	// find the function to call - all tool calls come here
	if funcall.Name == quarterlyResultsTools.FunctionDeclarations[0].Name {
		// check the params are populated
		ticker, exists := funcall.Args["ticker"]
		if !exists {
			log.Fatalln("Missing ticker")
		}
		year, exists := funcall.Args["year"]
		if !exists {
			log.Fatalln("Missing year")
		}
		quarter, exists := funcall.Args["quarter"]
		if !exists {
			log.Fatalln("Missing quarter")
		}
		// call the query database tool
		result = getResults(ticker.(string), year.(string), quarter.(string))
		log.Println("quarterly results result: " + result)
	} else {
		log.Println("unhandled function name: " + funcall.Name)
		return "", errors.New("unhandled function name: " + funcall.Name)
	}
	return result, nil
}

//////////////////////////////////////////
// client tools for external agents to use

// database agent client tool description
var CallQuarterlyResultsAgentTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{{
		Name:        "CallQuarterlyResultsAgent",
		Description: "Make a request to the quarterly results agent. The agent will extract the requested results file abd return it.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"message": {
					Type:        genai.TypeString,
					Description: "The natural language request message for the quarterly results agent",
				},
			},
			Required: []string{"message"},
		},
	}},
}

// client tool for the database agent
func CallQuarterlyResultsAgent(message string) (string, error) {
	log.Println("running callQuarterlyResultsAgent tool for :" + message)

	// get the agent endpoint
	hostname, ok := os.LookupEnv("QUARTERLY_RESULTS_AGENT_HOSTNAME")
	if !ok {
		log.Fatalln("environment variable QUARTERLY_RESULTS_AGENT_HOSTNAME not set")
	}
	port, ok := os.LookupEnv("QUARTERLY_RESULTS_AGENT_PORT")
	if !ok {
		log.Fatalln("environment variable QUARTERLY_RESULTS_AGENT_PORT not set")
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
	req, err := http.NewRequest("POST", "http://"+hostname+":"+port+"/agent", bytes.NewBuffer(reqDat))
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
