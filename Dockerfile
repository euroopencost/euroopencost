# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o eucost ./cmd/eucost

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/eucost .
COPY --from=builder /app/site ./site
COPY --from=builder /app/testdata ./testdata

# Standard-Port für Cloud Run
EXPOSE 8080

# Startet den Server
ENTRYPOINT ["./eucost", "serve", "--port", "8080"]
