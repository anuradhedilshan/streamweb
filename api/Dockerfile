FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /bin/api ./cmd/server

FROM alpine:3.21
RUN adduser -D app
USER app
COPY --from=build /bin/api /bin/api
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
