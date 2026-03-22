FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.work go.work
COPY shared/ shared/
COPY server/ server/
RUN cd server && go build -o /atk-server ./cmd/atk-server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /atk-server /usr/local/bin/atk-server
EXPOSE 8080
ENTRYPOINT ["atk-server"]
