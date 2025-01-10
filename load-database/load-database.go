package loaddatabase

import (
	"context"
	"log"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type tickerLine struct {
	Date  string `bson:"date"`
	Open  string `bson:"open"`
	High  string `bson:"high"`
	Low   string `bson:"low"`
	Close string `bson:"close"`
}

func LoadNasdaqDatabase(databaseName string) {

	// pull the db client
	mongodbUri, exists := os.LookupEnv("MONGODB_URI")
	if !exists {
		log.Fatalln("no MONGODB_URI in env vars")
	}
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongodbUri))
	if err != nil {
		log.Fatalln("error connecting to mongoDB", err)
	}

	// clear out any existing database and recreate
	err = client.Database(databaseName).Drop(context.TODO())
	if err != nil {
		log.Fatalln("error connecting to mongoDB", err)
	}

	// load the directory
	dir, exists := os.LookupEnv("NASDAQ_DATA")
	if !exists {
		log.Fatalln("no NASDAQ_DATA in env vars")
	}
	filenames, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalln("error reading dir", err)
	}

	// process each file
	for _, filename := range filenames {
		// open and read the content
		content, err := os.ReadFile(dir + filename.Name())
		if err != nil {
			log.Fatalln("error reading file:", filename.Name())
		}
		// split the content on new line
		contentLines := strings.Split(string(content), "\r\n")

		// create the collection based on the filename ticker
		coll := client.Database(databaseName).Collection(strings.Split(filename.Name(), ".")[0])

		// extract each line
		var dayData []interface{}
		for idx, line := range contentLines {
			// skip the first line
			if idx == 0 {
				continue
			}
			// populate the ticker line
			lineSplit := strings.Split(line, ",")
			if len(lineSplit) != 6 {
				// ignore - last line
				continue
			}
			dayData = append(dayData, tickerLine{
				Date:  lineSplit[1],
				Open:  lineSplit[2],
				High:  lineSplit[3],
				Low:   lineSplit[4],
				Close: lineSplit[5],
			})
		}

		// add to the collection as a batch
		_, err = coll.InsertMany(context.TODO(), dayData)
		if err != nil {
			log.Fatalln("error inserting batch:", err)
		}
	}

}
