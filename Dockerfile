FROM golang:latest

# create a directory inside the container to store the app. Make it the working directory
RUN mkdir -p /go/src/gowiki
WORKDIR /go/src/gowiki

# copy the gowiki directory (where the Dockerfile lives) into the container.
COPY . /go/src/gowiki

# download and install dependencies
RUN go get -d -v
RUN go install -v

# expose the PORT environment variable inside the container, then to the host so we can access it.
ENV PORT 8080
EXPOSE 8080

# build app
CMD ["go build"]

# Default command runs app
CMD ["./gowiki"]