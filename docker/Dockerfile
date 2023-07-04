FROM golang:1.20.5
LABEL authors="jameskozlowski"
# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY ./go.mod ./go.sum ./
RUN go mod download

# Copy the source code.
COPY ./ ./

# Build
RUN go build ./cmd/api

# expose port
EXPOSE 4000

# Run
CMD ["./api"]