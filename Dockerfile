# FROM golang:1.20

# WORKDIR /app

# COPY go.mod ./
# COPY go.sum ./
# RUN go mod download

# COPY . ./

# RUN go build -o sidekick .

# ENTRYPOINT ["./sidekick"]
# Build stage

FROM golang:1.20 AS build-stage

WORKDIR /app

# Copy go module files and download dependencies
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy the source code and build the application
COPY . ./
RUN CGO_ENABLED=0 go build -o sidekick .

# Final stage
FROM alpine:3.17 AS final-stage

# Add ca-certificates for SSL/TLS communication
RUN apk add --update --no-cache ca-certificates

# Correcting the path here
COPY --from=build-stage /app/sidekick /app/sidekick

# Create user for added security
RUN addgroup -S sidekick && adduser -u 1234 -S sidekick -G sidekick

# Switch to the new user
USER 1234

WORKDIR /app

# Set the entrypoint to the binary
ENTRYPOINT ["./sidekick"]
