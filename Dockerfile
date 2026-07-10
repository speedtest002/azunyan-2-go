FROM golang:1.23-alpine AS build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown

RUN CGO_ENABLED=0 go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -w -s" -o bot github.com/speedtest002/azunyan-2-go

FROM alpine

COPY --from=build /build/bot /bin/bot

ENTRYPOINT ["/bin/bot"]
