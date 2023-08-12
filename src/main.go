package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/joho/godotenv"
	"github.com/thedevsaddam/renderer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	//"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//initializing global var to be used outside of main func
/*
 The renderer package is used for rendering JSON responses.
 It simplifies the process of rendering JSON and allows custom templates
 for JSON responses.



*/
var rnd *renderer.Render

// for querying db
var db *mongo.Database

// mongo client var
var client *mongo.Client

const (
	collectionName string = "code-snippets"
	port           string = ":9000"
)

type (
	/*
	 the tags are used to provide additional information about how the struct fields should be serialized or deserialized when
	 working with JSON data. These tags are often used by
	 external libraries (such as JSON encoding/decoding libraries) to determine how to map the struct fields to JSON keys and values.

	 same with bson data

	*/
	//this is the model for the code-snippet collection which will be stored in the database and retrived from the database
	// All fields must start with Capital Letters
	CodeSnippetModel struct {
		ID          primitive.ObjectID `bson:"_id,omitempty"`
		CreatedAt   time.Time          `bson:"createAt"`
		SnippetName string             `bson:"snippetname"`
		Code        string             `bson:"code"`
	}
	//this is the response json type which will be sent to the client when retrived from database or from client (req.body) to be stored in db
	// All fields must start with Capital letters
	CodeSnippet struct {
		ID          string    `json:"id"`
		SnippetName string    `json:"snippetname"`
		Code        string    `json:"code"`
		CreatedAt   time.Time `json:"created_at"`
	}
)

// the init func is used for initializing the global var to be used outside the main func

//REGARDING context.TODO()
/*
The context package in Go is used to manage the lifecycle of operations,
 particularly in scenarios where operations might need to be canceled or timed out.
 In the context of MongoDB operations like connecting and querying, using a context
 can provide better control over these operations, especially in cases where
you want to ensure that resources are properly released, or you want to cancel an operation that's taking too long.


using contexts in MongoDB operations is a good practice. It provides you with the flexibility to add deadlines,
 cancellations, or other control mechanisms in the future without modifying the core logic of your functions.

using context.TODO() doesn't directly add any timeout or cancellation mechanism. However, it's a best practice to provide
 a context to the operation in case you want to add such mechanisms in the future.

If you find that the mongo.Connect operation is taking too much time
 and you want to add a timeout or cancellation mechanism, you would need to
  create a context with a timeout or a cancellation and pass it to the mongo.Connect function.
*/

func init() {
	rnd = renderer.New()

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environmental variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
	}

	// err type of error
	var err error
	client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	if err == nil {
		fmt.Printf("mongodb isrunning now")
	}

	db = client.Database("Code-Snippet-Manager") // Replace with your actual database name
}

func createSnippet(w http.ResponseWriter, r *http.Request) {
	//creating an instance of Codesnippet json struct
	var c CodeSnippet

	//we are going to be decoding the json recieved from the frontend to a struct type

	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		// remember rnd pckg is used to send json response and it collects 3 values
		// the writer intance, status code, message to be sent , the mssg is a map data structure
		// here we are sending the err
		rnd.JSON(w, http.StatusProcessing, err)
		// we are returning from the function since we are getting an err while decoding
		//so theres no need to conti ue the execution of the function
		return
	}

	// validating input

	if c.Code == "" && c.SnippetName == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message1": "the code input field is requested",
			"message2": "the snippet Name field is requested",
		})
		// return from func no need to continue execution of func
		return
	}

	// if input is okay
	// create/insert into database
	// converting into a bson data to be inputted into the mongodb database because only bson
	//is supported with mongodb
	cm := CodeSnippetModel{
		ID:          primitive.NewObjectID(),
		CreatedAt:   time.Now(),
		Code:        c.Code,
		SnippetName: c.SnippetName,
	}

	// storing the data into the database
	result, err := db.Collection(collectionName).InsertOne(context.TODO(), &cm)
	if err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to save Code Snippet",
			"error":   err,
		})
		return
	}

	fmt.Printf("the result after saving in database is = %s\n", result)

	// returning the inserted id  as json response

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "Snippet created successfully",
		//"snippet_id": cm.ID.Hex(),
		"snippet_id": result.InsertedID,
	})

}

func getSnippet(w http.ResponseWriter, r *http.Request) {

	// Get the snippet name from the URL parameter
	snippetName := chi.URLParam(r, "snippetName")

	// Create a filter to find the snippet by its name
	filter := bson.M{"snippetname": snippetName}

	// Create a variable to hold the result of the find operation the bson snippet model
	var foundSnippet CodeSnippetModel

	// decoding the snippet into a bson data, codeSnippetmodel because the findone will return a bson data
	if err := db.Collection(collectionName).FindOne(context.TODO(), filter).Decode(&foundSnippet); err != nil {
		rnd.JSON(w, http.StatusNotFound, renderer.M{
			"message": "Snippet not found",
			"error":   err,
		})
		return
	}

	// we are storing the found bson data into the codesnippet struct json data structure
	codesnippets := CodeSnippet{
		ID:          foundSnippet.ID.Hex(),
		CreatedAt:   foundSnippet.CreatedAt,
		Code:        foundSnippet.Code,
		SnippetName: foundSnippet.SnippetName,
	}

	// sending the struct data to the frontend
	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": codesnippets,
	})

}

func getAllSnippets(w http.ResponseWriter, r *http.Request) {
	// var to hold the res of all bson data found in the database to a slice since its multiple dats
	snippets := []CodeSnippetModel{}

	// The Find method returns a cursor to the query results and an error
	cursor, err := db.Collection(collectionName).Find(context.TODO(), bson.M{})
	if err != nil {
		//panic(err)
		rnd.JSON(w, http.StatusNotFound, renderer.M{
			"error": err,
		})
		return
	}

	//  retrieve all documents from the cursor using the All method.
	if err = cursor.All(context.TODO(), &snippets); err != nil {
		//panic(err)
		rnd.JSON(w, http.StatusNotFound, renderer.M{
			"message": "failed to fetch snippets",
			"error":   err,
		})
		return
	}
	// codeSnippet Struct json to be sent to the frontend
	snippetsList := []CodeSnippet{}
	// looping through the snippets slice bson struct to be converted to the json slice of struct
	for _, s := range snippets {
		snippetsList = append(snippetsList, CodeSnippet{
			ID:          s.ID.Hex(),
			SnippetName: s.SnippetName,
			CreatedAt:   s.CreatedAt,
			Code:        s.Code,
		})
	}

	// sending the struct slice of json to the frontend
	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": snippetsList,
	})

}

func updateSnippet(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("update function getting started")
	// getting the id of the snippet code that wants to updated
	idstr := strings.TrimSpace(chi.URLParam(r, "codeid"))
	fmt.Printf("update function getting started")
	//  convert the received id to a MongoDB ObjectID using primitive.ObjectIDFromHex(id).
	id, err := primitive.ObjectIDFromHex(idstr)
	if err != nil {
		// If the conversion fails (invalid ID), send a JSON response
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "The id is invalid",
		})
		return
	}

	// a var to store the json data body received from the frontend
	var s CodeSnippet

	// decoding the json data recived to a json struct type
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}

	// validating input

	if s.Code == "" && s.SnippetName == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message1": "the code input field is requested",
			"message2": "the snippet Name field is requested",
		})
		// return from func no need to continue execution of func
		return
	}

	// The filter is specifying that you want to match documents with
	// a specific _id field value. The id variable is used as the value for the _id field.
	filter := bson.D{{Key: "_id", Value: id}}

	/*
	   This line creates an update document using the bson.D type.
	    The update is using the $set operator to modify the value of a field. It specifies that you want to update the
	   the following
	*/
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "snippetname", Value: s.SnippetName}, {Key: "code", Value: s.Code}}}}

	result, err := db.Collection(collectionName).UpdateOne(context.TODO(), filter, update)
	if err != nil {
		// panic(err)
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to update ",
			"error":   err,
		})
		return
	}

	// When you run this file for the first time, it should print:
	// Number of documents replaced: 1
	fmt.Printf("Documents updated: %v\n", result.ModifiedCount)

	// returning data to the frontend
	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": "Snippet updated successfully",
	})

}

func deleteSnippet(w http.ResponseWriter, r *http.Request) {
	// getting the id of the snippet code that wants to deleted
	idstr := strings.TrimSpace(chi.URLParam(r, "id"))

	//  convert the received id to a MongoDB ObjectID using primitive.ObjectIDFromHex(id).
	id, err := primitive.ObjectIDFromHex(idstr)
	if err != nil {
		// If the conversion fails (invalid ID), send a JSON response
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "The id is invalid",
		})
		return
	}
	// id to be deleted
	filter := bson.D{{Key: "_id", Value: id}}

	result, err := db.Collection(collectionName).DeleteOne(context.TODO(), filter)
	if err != nil {

		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to delete snippet",
			"error":   err,
		})
		return

	}

	// When you run this file for the first time, it should print:
	// Documents deleted: 1
	fmt.Printf("Documents deleted: %d\n", result.DeletedCount)

	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": "Code Snippet deleted successfully",
	})

}

func main() {

	/*

		This is a defer statement. It means that the function passed as an argument (func() { ... }) will be executed
		right before the main function returns.
		 In this case, it's disconnecting the MongoDB client from the server.
		 If there's an error during disconnection, it will trigger a panic.

	*/
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	/*
	   This code creates a channel called stopChan and uses the signal package to notify
	   it when an interrupt signal (e.g., Ctrl+C) is received. This is used to gracefully shut down the server.
	*/

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	/*
	   Creates a new chi router and attaches a logger middleware to it to log all requests.
	   Then it registers a handler for the root URL path ("/") using the GET method, which is the homeHandler function.
	*/

	r := chi.NewRouter()
	// log all requests
	r.Use(middleware.Logger)
	//r.Get("/", homeHandler)

	// Mounts the subrouter returned by the todoHandlers() function under the "/todo" URL path.
	r.Mount("/code-snippets", snippetsHandlers())

	/*
		Creates an instance of http.Server with various settings,
		including the address to listen on (Addr), the router to handle requests (Handler), and timeout settings
	*/
	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	/*
		This starts a new goroutine (using go func() { ... }()) to listen and serve incoming HTTP requests.
		 It logs the start of the server and handles any errors that might occur during the server's execution.
	*/
	go func() {
		log.Println("Listening on port ", port)
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()

	/*
	   (<-stopChan) waits for a signal to be received on stopChan, which happens when the interrupt signal is triggered (e.g., Ctrl+C).
	    When the signal is received, it triggers a graceful shutdown process. It creates a context with a timeout of 5 seconds,
	   attempts to gracefully shut down the server using srv.Shutdown(ctx), and logs the successful server shutdown.

	*/

	<-stopChan
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()
	log.Println("Server gracefully stopped!")
}

/*
The snippetsHandlers() function returns an http.Handler (which is a router) for managing routes related to todos.

	It creates a subrouter using chi.NewRouter(), groups the routes using rg.Group(...),

and maps each HTTP method to its corresponding handler function.
*/
func snippetsHandlers() http.Handler {
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router) {
		r.Get("/", getAllSnippets)
		r.Get("/{snippetName}", getSnippet)
		r.Post("/", createSnippet)
		r.Put("/{codeid}", updateSnippet)
		r.Delete("/{id}", deleteSnippet)
	})
	return rg
}

// This is a utility function that checks if an error occurred and,
// if so, logs it as a fatal error, which usually terminates the application.
func checkErr(err error) {
	if err != nil {
		log.Fatal(err) //respond with error page or message
	}

}
