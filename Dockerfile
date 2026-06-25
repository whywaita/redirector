# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /redirector .

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /redirector /redirector

EXPOSE 8080

ENTRYPOINT ["/redirector"]
