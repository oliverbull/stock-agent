package stockmarketinfoapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	datacombineagent "stock-agent/data-combine-agent"
	agentassemble "stock-agent/gemini-agent-assemble"

	"github.com/google/generative-ai-go/genai"
)

/////////////////
// data combine agent

// client agent tools descriptions from the agents
var stockMarketInfoTools = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{
		datacombineagent.CallDataCombineAgentTool.FunctionDeclarations[0],
	},
}

// agent initialization
func InitStockMarketInfoAgent(ctx context.Context) (*agentassemble.Agent, error) {
	system := `
You are an AI agent that can respond to natural language requests for stock market data information.
You have access to underlying agent tools that can perform the following actions:
* Take compound requests and use the tools it has access to for result generation.
* Get the daily nasdaq stock market data for open, close, high, low.
* Get a company quarterly results release with their financial data.
Based on the request, think about how to approach this problem, then act by performing necessary actions (like calling tools), and finally observe the results to refine your understanding and provide a final answer
You can call the tools multiple times to get the answer to the request.
You can call the same tool multiple times to get the answer to the request.
When you know the final answer, you must start the response with the words 'Final Answer:'
`
	// initialize the agent
	var tools = []*genai.Tool{stockMarketInfoTools}
	agentStockMarketInfo, err := agentassemble.InitAgent(ctx, &system, tools, callStockMarketInfoTool)
	if err != nil {
		log.Println("Error initializing the database agent")
		return nil, err
	}

	// always start a new session
	agentStockMarketInfo.NewSession()

	return agentStockMarketInfo, err
}

// tool call handler
func callStockMarketInfoTool(funcall genai.FunctionCall) (string, error) {
	result := ""
	var err error
	// find the function to call - all tool calls come here
	if funcall.Name == stockMarketInfoTools.FunctionDeclarations[0].Name {
		// check the params are populated
		message, exists := funcall.Args["message"]
		if !exists {
			err := errors.New("missing arg: message")
			log.Println(err)
			return err.Error(), err
		}
		// call the query database tool
		datacombineagent.CallDataCombineAgent(message.(string))
		result, err = datacombineagent.CallDataCombineAgent(message.(string))
		if err != nil {
			log.Println("CallDataCombineAgent():", err)
			return err.Error(), err
		}
		log.Println("call data combine results result: " + result)
	} else {
		log.Println("unhandled function name: " + funcall.Name)
		return "", errors.New("unhandled function name: " + funcall.Name)
	}
	return result, nil
}

//////////////////////////////////////////
// client tools for external agents to use

// stock market info app (agent) client tool description
var CallStockMarketInfoAppTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{{
		Name:        "callStockMarketInfoApp",
		Description: "Make a request to the stock market info app. The app will process the request and return the result.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"message": {
					Type:        genai.TypeString,
					Description: "The natural language request message for the app",
				},
			},
			Required: []string{"message"},
		},
	}},
}

// client tool for the stock market app agent
func CallStockMarketInfoApp(message string) (string, error) {
	log.Println("running CallStockMarketInfoApp tool for :" + message)

	// get the agent endpoint
	hostname, ok := os.LookupEnv("STOCK_MARKET_INFO_APP_HOSTNAME")
	if !ok {
		err := errors.New("environment variable STOCK_MARKET_INFO_APP_HOSTNAME not set")
		log.Println(err)
		return "", err
	}
	port, ok := os.LookupEnv("STOCK_MARKET_INFO_APP_PORT")
	if !ok {
		err := errors.New("environment variable STOCK_MARKET_INFO_APP_PORT not set")
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
