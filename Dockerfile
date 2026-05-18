FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/iec-104-cli .

FROM alpine:3.21

RUN adduser -D -H -u 10001 iec104
COPY --from=build /out/iec-104-cli /usr/local/bin/iec-104-cli

USER iec104
ENTRYPOINT ["iec-104-cli"]
CMD ["help"]
