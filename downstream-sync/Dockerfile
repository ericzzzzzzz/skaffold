FROM golang:1.21 AS builder

# Set necessary environment variables for building
ARG CGO_ENABLED=0
ENV GO111MODULE=on

# Set work directory
WORKDIR /app

# Copy project files and dependencies
COPY . .

RUN go build -o app-server server/main.go
RUN go build -o app-connect connect/main.go
RUN go build -o app-client client/main.go


FROM alpine:3
COPY --from=builder /app/app-server /app/app-connect /app/app-client .

CMD ["cp", "app-server", "app-connect", "app-client", "/abccc"]
