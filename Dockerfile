# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26-alpine3.24 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN if [ "$TARGETARCH" = "arm" ]; then export GOARM="${TARGETVARIANT#v}"; fi \
    && CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
       go build -trimpath -ldflags="-s -w" -o /out/sublinkX .

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /app/db /app/logs /app/template

WORKDIR /app

ENV TZ=Asia/Shanghai

COPY --from=builder /out/sublinkX /app/sublinkX

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/app/sublinkX", "healthcheck"]

ENTRYPOINT ["/app/sublinkX"]
