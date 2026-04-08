FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -trimpath -o tsk9s .

# Download k9s
ARG K9S_VERSION=v0.50.6
RUN wget -qO- "https://github.com/derailed/k9s/releases/download/${K9S_VERSION}/k9s_Linux_${TARGETARCH}.tar.gz" | tar xz -C /usr/local/bin k9s

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

COPY --from=build /app/tsk9s /usr/local/bin/tsk9s
COPY --from=build /usr/local/bin/k9s /usr/local/bin/k9s

VOLUME /data

ENTRYPOINT ["tsk9s", "--state-dir", "/data/tsk9s-state"]
