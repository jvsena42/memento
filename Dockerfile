## Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency files first for better layer caching
# These only changes when adding/ removing dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# BUild the bynary
# CGO_ENABLED=0 ensures a full static binary, with no C dependencies
# -o memento puts the output binary in /app/memento
RUN CGO_ENABLED=0 go build -o memento ./cmd/memento

## Run
## Start fres with a tiny image
FROM alpine:3.19

## Add CA certificates so HTTPS call to Twitter API works
RUN apk add --no-cache ca-certificates

## Create a non-root user for security
RUN adduser -D -h /app appuser
WORKDIR /app

# Copy only what we need from the builder stage
COPY --from=builder /app/memento .
COPY --from=builder /app/migrations ./migrations

# Dont run as root
USER appuser

# What runs when the container starts
ENTRYPOINT ["./memento"] 