# builder image
FROM golang:1.23-bullseye as builder

ENV GOPROXY=https://goproxy.cn,direct
ENV GOPRIVATE=git.fiblab.net
RUN apt-get update && apt-get install -y git libproj-dev \
    && apt-get clean && rm -rf /var/lib/apt/lists/* \
    && go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /build
COPY . /build
RUN swag init && GOOS=linux go build -a -o backend .

# generate clean, final image for end users
FROM debian:bullseye-slim
COPY --from=builder /build/backend .
RUN apt-get update && apt-get install -y libproj-dev \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
ENV GIN_MODE=release
# executable
ENTRYPOINT [ "./backend" ]
