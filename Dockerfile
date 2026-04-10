FROM golang:1.24 AS build-env
WORKDIR /app

RUN apt-get update && apt-get install -y gcc libc6-dev libx11-dev pkg-config && rm -rf /var/lib/apt/lists/*

COPY ./src/go.mod ./src/go.sum ./
RUN go mod download

# Copy the rest of the Go source code from src
COPY ./src .

RUN go build -o main ./*.go

# Final stage for the smaller runtime image
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y ca-certificates openssl && rm -rf /var/lib/apt/lists/*

RUN addgroup --system --gid 1001 owl && adduser --system --uid 1001 owl

WORKDIR /app

COPY --from=build-env /app/main .

RUN chown -R owl:owl /app

USER 1001

EXPOSE 3000
ENV PORT=3000

ENTRYPOINT ["./main", "--serve"]
