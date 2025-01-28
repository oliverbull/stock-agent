package datacombineagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	databaseagent "stock-agent/database-agent"
	agentassemble "stock-agent/gemini-agent-assemble"
	quarterlyresultsagent "stock-agent/quarterly-results-agent"

	"github.com/google/generative-ai-go/genai"
)

/////////////////
// data combine agent

// client agent tools descriptions from the agents
var dataCombineTools = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{
		quarterlyresultsagent.CallQuarterlyResultsAgentTool.FunctionDeclarations[0],
		databaseagent.CallDatabaseAgentTool.FunctionDeclarations[0],
	},
}

// agent initialization
func InitDataCombineAgent(ctx context.Context) (*agentassemble.Agent, error) {
	system := `
You are an AI agent that can process and answer requests on nasdaq companies.
You have access to underlying agent tools that can perform the following actions:
* Get the daily nasdaq stock market data for open, close, high, low.
* Get a company quarterly results release with their financial data.
Based on the request, think about how to approach this problem, then act by performing necessary actions (like calling tools), and finally observe the results to refine your understanding and provide a final answer
You can call the tools multiple times to get the answer to the request.
You can call the same tool multiple times to get the answer to the request.
When you know the final answer, you must start the response with the words 'Final Answer:'
`
	// initialize the agent
	var tools = []*genai.Tool{dataCombineTools}
	agentDataCombine, err := agentassemble.InitAgent(ctx, &system, tools, callDataCombineTool)
	if err != nil {
		log.Println("Error initializing the database agent")
		return nil, err
	}

	// always start a new session
	agentDataCombine.NewSession()

	return agentDataCombine, err
}

// tool call handler
func callDataCombineTool(funcall genai.FunctionCall) (string, error) {
	result := ""
	var err error
	// find the function to call - all tool calls come here
	if funcall.Name == dataCombineTools.FunctionDeclarations[0].Name {
		// check the params are populated
		message, exists := funcall.Args["message"]
		if !exists {
			err := errors.New("missing arg: message")
			log.Println(err)
			return err.Error(), err
		}
		// call the query database tool
		result, err = quarterlyresultsagent.CallQuarterlyResultsAgent(message.(string))
		if err != nil {
			log.Println("CallQuarterlyResultsAgent():", err)
			return err.Error(), err
		}
		log.Println("call quarterly results result: " + result)
	} else if funcall.Name == dataCombineTools.FunctionDeclarations[1].Name {
		// check the params are populated
		message, exists := funcall.Args["message"]
		if !exists {
			err := errors.New("missing arg: message")
			log.Println(err)
			return err.Error(), err
		}
		// call the query command
		result, err = databaseagent.CallDatabaseAgent(message.(string))
		if err != nil {
			log.Println("CallDatabaseAgent():", err)
			return err.Error(), err
		}
		log.Println("call database result: " + result)
	} else {
		log.Println("unhandled function name: " + funcall.Name)
		return "", errors.New("unhandled function name: " + funcall.Name)
	}
	return result, nil
}

//////////////////////////////////////////
// client tools for external agents to use

// data combine agent client tool description
var CallDataCombineAgentTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{{
		Name:        "callDataCombineAgent",
		Description: "Make a request to the data combine agent. The agent will process the requested query using the tools it can access and return the combined result.",
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
func CallDataCombineAgent(message string) (string, error) {
	log.Println("running CallDataCombineAgent tool for :" + message)

	// get the agent endpoint
	hostname, ok := os.LookupEnv("DATA_COMBINE_AGENT_HOSTNAME")
	if !ok {
		err := errors.New("environment variable DATA_COMBINE_AGENT_HOSTNAME not set")
		log.Println(err)
		return "", err
	}
	port, ok := os.LookupEnv("DATA_COMBINE_AGENT_PORT")
	if !ok {
		err := errors.New("environment variable DATA_COMBINE_AGENT_PORT not set")
		log.Println(err)
		return "", err
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
