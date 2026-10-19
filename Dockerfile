FROM golang:1.27-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/importer ./cmd/worker

FROM alpine:3.20

WORKDIR /app

COPY --from=build /bin/importer /bin/importer

RUN addgroup -S app && adduser -S app -G app
USER app

ENTRYPOINT ["/bin/importer"]