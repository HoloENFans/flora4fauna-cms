FROM golang:alpine AS build

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o pocketbase .

FROM alpine:latest

WORKDIR /pb

COPY --from=build /build/pocketbase /pb/pocketbase
COPY pb_migrations /pb/pb_migrations

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

EXPOSE 8090

ENTRYPOINT ["/pb/pocketbase", "serve", "--http=0.0.0.0:8090", "--dir=/pb_data", "--publicDir=/pb_public"]