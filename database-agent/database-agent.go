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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// database schema for each ticker collection
type tickerLine struct {
	Date  string `bson:"date" json:"date"`
	Open  string `bson:"open" json:"open"`
	High  string `bson:"high" json:"high"`
	Low   string `bson:"low" json:"low"`
	Close string `bson:"close" json:"close"`
}

/////////////////
// database agent

// specific data range query database tool description
// var queryDatabaseTool = &genai.Tool{
var databaseTools = &genai.Tool{
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
	},
		{
			Name:        "commandQueryDatabase",
			Description: "Run the supplied MongoDB command on the nasdaq database",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"command": {
						Type:        genai.TypeString,
						Description: "The generated MongoDB command in JSON format",
					},
				},
				Required: []string{"command"},
			},
		},
	},
}

// specific data range query database tool
func queryDatabase(ticker string, startDate string, endDate string) string {
	log.Println("running queryDatabase tool for " + ticker + " with date range " + startDate + " - " + endDate)

	// connect a client to the database
	mongodbUri, exists := os.LookupEnv("MONGODB_URI")
	if !exists {
		log.Println("missing datbase URI in env vars")
		return "missing datbase URI in env vars, cannot continue"
	}
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongodbUri))
	if err != nil {
		log.Println("mongo connect() error:", err)
		return "mongo connect() error:" + err.Error()
	}
	defer client.Disconnect(context.TODO())

	// get the collection
	coll := client.Database("nasdaq").Collection(ticker)
	if coll == nil {
		log.Println("empty collection for ticker: " + ticker)
		return "empty collection for ticker: " + ticker + ", cannot continue"
	}

	// prep the filter and find
	filter := bson.D{{"date", bson.D{{"$gte", startDate}, {"$lte", endDate}}}}
	cursor, err := coll.Find(context.TODO(), filter)
	if err != nil {
		log.Println("coll.Find() error:", err)
		return "coll.Find() error:" + err.Error()
	}

	// unpack the cursor into a slice and then a string
	var results []tickerLine
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Println("cursor.All() error:", err)
		return "cursor.All() error:" + err.Error()
	}
	resultsStr, err := json.Marshal(results)
	if err != nil {
		log.Println("json.Marshal() error:", err)
		return "json.Marshal() error:" + err.Error()
	}

	return string(resultsStr)
}

// open command query query database tool
func commandQueryDatabase(command string) string {
	log.Println("running commandQueryDatabase tool for " + command)

	// connect a client to the database
	mongodbUri, exists := os.LookupEnv("MONGODB_URI")
	if !exists {
		log.Println("missing datbase URI in env vars")
		return "missing datbase URI in env vars, cannot continue"
	}
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongodbUri))
	if err != nil {
		log.Println("mongo connect() error:", err)
		return "mongo connect() error:" + err.Error()
	}
	defer client.Disconnect(context.TODO())

	// get the nasdaq db
	db := client.Database("nasdaq")
	if db == nil {
		log.Println("empty database")
		return "empty database, cannot continue"
	}

	// run the command
	result := db.RunCommand(context.TODO(), command)
	resultRaw, err := result.Raw()
	if err != nil {
		log.Println("runcommand raw error:", err)
		return "runcommand raw error:" + err.Error()
	}

	return resultRaw.String()
}

// agent initialization
func InitDatabaseAgent(ctx context.Context) (*agentassemble.Agent, error) {
	// check there is a uri for the db
	_, exists := os.LookupEnv("MONGODB_URI")
	if !exists {
		err := errors.New("missing MONGODB_URI in env vars")
		log.Println(err)
		return nil, err
	}
	system := `
You are an AI agent that can perform MongoDB database queries.
You have access to the underlying database through the query and command tools.
The database contains daily nasdaq stock market data.
You MUST use the tools to help answer the request and retrun the result.
`
	// initialize the agent
	//var tools = []*genai.Tool{queryDatabaseTool, commandQueryDatabaseTool}
	var tools = []*genai.Tool{databaseTools}
	agentDatabase, err := agentassemble.InitAgent(ctx, &system, tools, callDatabaseTool)
	if err != nil {
		log.Println("Error initializing the float agent")
		return nil, err
	}

	// always start a new session
	agentDatabase.NewSession()

	return agentDatabase, err
}

// tool call handler
func callDatabaseTool(funcall genai.FunctionCall) (string, error) {

	result := ""
	// find the function to call - all tool calls come here
	if funcall.Name == databaseTools.FunctionDeclarations[0].Name {
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
		// call the query database tool
		result = queryDatabase(ticker.(string), startDate.(string), endDate.(string))
		log.Println("query database result: " + result)
	} else if funcall.Name == databaseTools.FunctionDeclarations[1].Name {
		// check the params are populated
		command, exists := funcall.Args["command"]
		if !exists {
			log.Fatalln("Missing command")
		}
		// call the query command
		result = commandQueryDatabase(command.(string))
		log.Println("command query database result: " + result)
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
