FROM golang:1.22 AS builder
LABEL authors="alan"
WORKDIR /app
# Copy a local Mars directory into the image
#COPY . .

# Clone the latest Mars version
RUN git config --global http.sslVerify false
RUN git clone https://gitlab.bedge/whoKnows/Mars.git .

# Copy a conf.json file into the container
COPY conf.json known_entities.json conf/
COPY certs/ conf/certs/
RUN ls conf
RUN ls conf/certs
RUN ls conf/certs/1

# Download the go modules & build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o Mars

# Verify all files are present
RUN ls -la /app

FROM ubuntu:latest AS runner

# Expose the ports
EXPOSE 3001
EXPOSE 4567
EXPOSE 45671
WORKDIR /app

# Copy over the necessary files from the builder
RUN mkdir -p /app/conf /app/frontend
COPY --from=builder /app/Mars /app/Mars
COPY --from=builder /app/conf/* /app/conf/
COPY --from=builder /app/frontend/ /app/frontend/

# Verify the files are present
RUN ls /app/conf
RUN ls /app/frontend
RUN cat /app/conf/conf.json

# Start Mars
ENTRYPOINT ["/app/Mars"]
