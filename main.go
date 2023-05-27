package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Articles struct {
	ID         primitive.ObjectID `bson:"_id"`
	Dj_user_id string             `bson:"dj_user_id"`
	Title      string             `bson:"title"`
	Contents   string             `bson:"contents"`
	Date       string             `bson:"date"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	var uri string
	if uri = os.Getenv("MONGODB_URI"); uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environmental variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	collections := client.Database("dj_board").Collection("articles")
	filter := bson.D{{"title", "윤석열 vs 이재명"}}

	var result Articles
	err = collections.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		fmt.Println("err FindOne")
	}
	output, err := json.MarshalIndent(result, " ", "	")
	if err != nil {
		fmt.Println("err json MarshalIndent")
	}
	fmt.Println(string(output))
}
