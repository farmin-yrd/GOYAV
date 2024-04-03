FROM golang:1.22-alpine3.19 AS dependency
COPY go.mod go.sum /src/
WORKDIR /src
RUN go mod download

FROM dependency AS build
COPY . /src
WORKDIR /src
RUN go build -C ./cmd  -o /bin/service

# FROM scratch
FROM alpine:3.19
COPY --from=build /bin/service /bin/service
EXPOSE 80
ENTRYPOINT [ "/bin/service" ]
