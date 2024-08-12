FROM golang:1.22 AS builder
LABEL authors="alan"
WORKDIR /app

# Clone the latest Mars version
RUN git config --global http.sslVerify false
RUN git clone https://gitlab.bedge/whoKnows/enceladus.git .

# Download the go modules & build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o enceladus

FROM ubuntu:latest AS runner

# Expose the port
WORKDIR /app

# Copy Enceledus from the builder
COPY --from=builder /app/enceladus /app/enceladus
COPY --from=builder /app/frontend/ /app/frontend/

# Start Enceladus
ENTRYPOINT ["/app/enceladus"]