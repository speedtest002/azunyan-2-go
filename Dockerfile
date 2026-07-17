# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /build

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG VERSION=dev
ARG COMMIT=unknown

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOARM=${TARGETVARIANT#v} \
    go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -w -s" \
    -o bot github.com/speedtest002/azunyan-2-go

FROM alpine
RUN apk add --no-cache tzdata
COPY --from=build /build/bot /bin/bot
ENTRYPOINT ["/bin/bot"]
