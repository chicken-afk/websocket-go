# Use official Go image
FROM golang:1.21

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum for dependency management
COPY go.mod go.sum ./

# Download dependencies (if any)
RUN go mod tidy

# Copy the entire application
COPY . .

# Run the application
CMD ["go", "run", "*.go"]