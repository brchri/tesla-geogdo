FROM golang:1.21-alpine3.18 AS builder

ARG BUILD_VERSION
ARG BUILD_HASH

RUN test -n "${BUILD_VERSION}" || (echo "Build argument BUILD_VERSION is required but not provided" && exit 1)

WORKDIR /app
COPY . ./

RUN cp examples/config.multiple.yml ./config.example.yml && \
    go test ./... && \
    go build -ldflags="-X main.version=${BUILD_VERSION} -X main.commitHash=${BUILD_HASH:0:7}" -o tesla-geogdo cmd/app/main.go

FROM alpine:3.18

ARG USER_UID=10000
ARG USER_GID=$USER_UID

VOLUME [ "/app/config" ]
WORKDIR /app

RUN apk add --no-cache bash tzdata && \
    addgroup --gid $USER_GID nonroot && \
    adduser --uid $USER_UID --ingroup nonroot --system --shell bin/bash nonroot && \
    chown -R nonroot:nonroot /app

COPY --from=builder --chown=nonroot:nonroot --chmod=755 /app/tesla-geogdo /app/config.example.yml /app/

ENV PATH="/app:${PATH}"

USER nonroot

CMD [ "/app/tesla-geogdo", "-c", "/app/config/config.yml" ]
