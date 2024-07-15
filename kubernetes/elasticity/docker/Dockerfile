FROM golang:1.22-alpine
WORKDIR /app

# Add bash to alpine
RUN apk add --no-cache bash curl

# Make the out directory
RUN mkdir -p /app/out

COPY go.* .

RUN go mod download

COPY . .

# Build app
RUN ./make.sh

ENTRYPOINT ["/app/run.sh"]
