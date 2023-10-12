package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os" //This is Nice
    "fmt"
	"bytes"
	"time"
	"strconv"
	"github.com/gorilla/mux" //	This is for creating meaningful routes
	_ "github.com/lib/pq"
	//	For making connection to the Postgres{Maybe inside docker}
)

//Exported Job Profile

// ID can be simple int++
// timestamp is maybe in epoch or in milliseconds ? Need some clever solution for that maybe
// IsDone will tell basically is executed or not.
type Job struct {
	ID        int    `json:"id"`
	Timestamp string `json:"ts"`
	IsDone    bool   `json:"isdone"`
}

// Just for testing purpose
// --> This function got trigger when user use the value /hello
// func Hello(w http.ResponseWriter, r *http.Request) {

// 	//This Line is basically a response Writer. Can Send HTML content also.
// 	w.Write([]byte("Hello-Vikas, I Am Working this time"))
// }

// This is for adding Middleware to the response that's all.
func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application-json")
		next.ServeHTTP(w, r)
	})
}

//===============================================================================================
//							Subroutine call to the client
//===============================================================================================

//When Any user do a update, create or delete a job I want to send a signal to the client so that
// whatever this kind of operation happens, he(client) is have to look again to the database.
// and this is async call, with mandatory response requirement. So it will send a signal to client,
// and send it again if it didn't get the ack. and again and again. Let's say 20 times in 5 minutes.
// ? and maybe decide that client is not connected.

func makeschedule(databaselink *sql.DB,job Job,seconds int64) bool{
	fmt.Println("Scheduling this after %d seconds",seconds)
	time.Sleep(time.Duration(seconds) * time.Second)
	return signalClient(job)
}
func updatestatusofjob(reply bool,database *sql.DB,job Job){
	if reply == false{
		fmt.Println("Client was not able to perform the job")
		return
	}
	_, err := database.Exec("update job_table SET isdone = true where id = $1", job.ID)
	if err != nil {
		log.Fatal(err)
	}
}

//decide when to send the signal to the client.
func decidewhentosendsignal(databaselink *sql.DB,job Job){
	Timestamp 	:= job.Timestamp
	seconds,err := howmuchfutureinseconds(Timestamp)
	if err != nil {
		fmt.Println("Not Sending the request to client, Not able to parse Timestamp")
		return
	}
	if seconds ==0{
		fmt.Println("Not Sending the request to client, Since Timestamp Expired")
		return
	}
    ReplyfromClient := makeschedule(databaselink , job , seconds )
	updatestatusofjob(ReplyfromClient,databaselink,job)
	//now on the basis of reply from client, decide changing the parameters.
}
func howmuchfutureinseconds(timestamp string)(int64,error){
	timestampMilliseconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return 0, err
	}
	// Convert the timestamp to a time.Time object
	timestampTime := time.Unix(0, timestampMilliseconds*int64(time.Millisecond))

	// Get the current time
	currentTime := time.Now()

	// Calculate the time difference in seconds
	diff := timestampTime.Sub(currentTime)
	secondsDifference := int64(diff.Seconds())

	return secondsDifference, nil
}

//this function is will check, if we can accept a job for our db.
func isthisfuturetime(timeInMilliseconds string) bool{
	// Convert the time in milliseconds to an integer
	milliseconds, err := strconv.ParseInt(timeInMilliseconds, 10, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return false // You can choose the appropriate behavior for your use case
	}
	t := time.Unix(0, milliseconds*int64(time.Millisecond))
	currentTime := time.Now()
	return t.After(currentTime)
}


func signalClient(job Job) bool{
    fmt.Println("---------------------------------")
    fmt.Println("----SENDING REQUEST TO CLIENT----")
    fmt.Println("---------------------------------")
    requestBody, err := json.Marshal(job)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return false
	}
	url 			 := "http://192.168.1.6:5299/doexecute"
	fmt.Println("Sending Body of %v",requestBody)
	resp, err := sendPOSTRequest(url, requestBody)
	if err != nil {
		fmt.Println("Error:", err)  // This has to be fixed.
		return false
	}
	defer resp.Body.Close()

		// Check the response
		if resp.StatusCode == http.StatusOK {
			fmt.Println("Request was successful")
			return true

		} else {
			fmt.Println("Request failed with status code:", resp.Status)
			return false
		}
}

func sendPOSTRequest(url string, body []byte) (*http.Response, error) {
    // Create an HTTP request
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }

    // Set the HTTP version to 1.1 and add headers as needed
    req.Proto = "HTTP/1.1"
    req.Header.Add("Content-Type", "application/json")
    // Add other headers as required

    // Send the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }

    return resp, nil
}
//------------------------------------------------------------------------------------------------
//							Create a New Job { Route Request Handler }
//------------------------------------------------------------------------------------------------

//@ table with all the data is job_table

func getjobs(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("Select * from job_table")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		jobs := []Job{}
		for rows.Next() {
			var j Job
			if err := rows.Scan(&j.ID, &j.Timestamp, &j.IsDone); err != nil {
				log.Fatal(err)
			}
			jobs = append(jobs, j)
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(w).Encode(jobs)

	}
}

// This will give the Job Data on the basis of any ID
func getjob(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		var job Job
		err := db.QueryRow("Select * from job_table where id = $1", id).Scan(&job.ID, &job.Timestamp, &job.IsDone)
		if err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(w).Encode(job)
	}
}

// Create a New job

func createjob(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var job Job
		json.NewDecoder(r.Body).Decode(&job)
		Flag := isthisfuturetime(job.Timestamp)
		if job.IsDone{
			http.Error(w, "Marking true is the server work, Please make it false and try again.", http.StatusBadRequest)
			return
		}
		if !Flag {
			http.Error(w, "Past Jobs can't be accepted", http.StatusBadRequest)
			return
		}
		err := db.QueryRow("INSERT into job_table (timestamp, isdone) values ($1 ,$2) RETURNING id", job.Timestamp, job.IsDone).Scan(&job.ID)
		if err != nil {
			log.Fatal(err)
		}
		go decidewhentosendsignal(db,job)  // The Async Call to call the client.
		json.NewEncoder(w).Encode(job)
	}
}

// Update Job
func updatejob(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var job Job
		json.NewDecoder(r.Body).Decode(&job)
		Flag := isthisfuturetime(job.Timestamp)
		if job.IsDone{
			http.Error(w, "Marking true is the server work, Please make it false and try again.", http.StatusBadRequest)
			return
		}
		if !Flag {
			http.Error(w, "Past Jobs can't be accepted", http.StatusBadRequest)
			return
		}
		vars := mux.Vars(r)
		id := vars["id"]
		_, err := db.Exec("update job_table SET timestamp = $1, isdone = $2 where id = $3", job.Timestamp, job.IsDone, id) // Is this correct representation ?
		if err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(w).Encode(job)
	}
}

// Delete Job
func deletejob(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		_, err := db.Exec("DELETE from job_table where id = $1", id)
		if err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(w).Encode("Job Deleted, Thanks for removing !")
	}
}

func main() {
	// http.HandleFunc("/hello", Hello)
	// log.Fatal(http.ListenAndServe(":5100", nil))

	//New code for Dev
	//----> Connection to DB <-----
	// Opening a driver typically will not attempt to connect to the database.
	// pool, err = sql.Open("driver-name", *dsn)

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// create the table if not exist
	_, err = db.Exec("create table if not exists job_table (id SERIAL PRIMARY KEY, timestamp TEXT, isdone bool)")
	if err != nil {
		log.Fatal(err)
	}
	// Took inspiration for the courier-server Repo Basically
	//New Router
	router := mux.NewRouter()

	//Routing Logic
	router.HandleFunc("/getjobs", getjobs(db)).Methods("Get")
	router.HandleFunc("/getjob/{id}", getjob(db)).Methods("Get") // Id is integer in this case
	router.HandleFunc("/create", createjob(db)).Methods("Post")
	router.HandleFunc("/update/{id}", updatejob(db)).Methods("Post") //This {id} is in the request itself,cool.
	router.HandleFunc("/delete/{id}", deletejob(db)).Methods("Post")

	// ** NOTE TO SELF, HOW TO Introduce producer consumer here ?

	log.Fatal(http.ListenAndServe(":5199", jsonContentTypeMiddleware(router)))
}
