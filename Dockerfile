# use official Golang image
FROM golang:1.16.3-alpine3.13

#The Working DIR
WORKDIR /app

#copy the source code
COPY .  .

#download and install the dependencies
RUN go get -d -v ./...

#IS this correct Way, No Idea but it is working for now.
WORKDIR /app/radar
#BUILD the go app
RUN go build -o serverarium

#expose the port
EXPOSE 5199

#rum the Executable
CMD ["./serverarium"]