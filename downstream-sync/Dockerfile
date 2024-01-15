FROM golang:alpine AS builder

# Set necessary environment variables for building
ARG CGO_ENABLED=0
ENV GO111MODULE=on

# Set work directory
WORKDIR /app

# Copy project files and dependencies
COPY . .

RUN go build -o server server/main.go
RUN go build -o connect/main.go

CMD ["./server"]
