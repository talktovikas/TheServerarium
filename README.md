# TheServerarium
This will be to provide the server side data.
Follow which The Serverarium will them them to do

How to use
```
docker-compose build
docker-compose up
```
One can use postman and use /getjobs to fetch saved jobs in the database(postgresql)
One can create a job by sending body in json
```
{
  "ts":"1696974475"
   "isdone":false
}

```

all other routes can be used to do crud operations on database.
have to strengthen communication channel to the client.
