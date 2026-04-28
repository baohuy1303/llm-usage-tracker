# Stage 1: Build the API binary in src directory
# Name the stage as builder
FROM golang:1.26-alpine AS builder
WORKDIR /src

# Cache dependency download as its own layer
COPY go.mod go.sum ./
RUN go mod download

# Copy into src directory
# CGO_ENABLED=0 works because modernc.org/sqlite is pure Go.
COPY . .

# Tell it to build in /out directory, taken the source from ./cmd/api folder
# The final binary file is named api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

# Stage 2: Minimal runtime image
# starts a tiny image
FROM alpine:latest
# go to /app directory
WORKDIR /app
# go back to builder stage and copy the binary from /out/api directory
COPY --from=builder /out/api .

EXPOSE 8080
# Copied the compiled binary, now run it (recall the binary name is api)
CMD ["./api"]
