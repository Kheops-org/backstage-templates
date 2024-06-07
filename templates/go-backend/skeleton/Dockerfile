# Stage 1: Build
FROM golang:1-bookworm as build

# Set the working directory to /sreez
WORKDIR /sreez

# Copy the Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY *.go .
COPY docs/ docs/

# Build the executable
RUN go generate
RUN go build -o run main.go

# Stage 2: Production
FROM golang:1-bookworm

# Set the working directory to /sreez
WORKDIR /sreez

# Copy the executable from the build stage
COPY --from=build /sreez/run .
COPY --from=build /sreez/docs docs
# Expose the port
EXPOSE 8080

# Run the executable when the container starts
CMD ["./run"]
