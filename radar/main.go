package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os" //This is Nice

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
		err := db.QueryRow("INSERT into job_table (timestamp, isdone) values ($1 ,$2) RETURNING id", job.Timestamp, job.IsDone).Scan(&job.ID)
		if err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(w).Encode(job)
	}
}

// Update Job
func updatejob(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var job Job
		json.NewDecoder(r.Body).Decode(&job)
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
