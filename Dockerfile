# Build stage
FROM golang:1.22 as builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /droneops-sim ./cmd/droneops-sim

# Runtime stage using Distroless (minimal and secure)
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /droneops-sim /droneops-sim

ENTRYPOINT ["/droneops-sim"]