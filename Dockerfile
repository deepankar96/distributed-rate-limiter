# syntax=docker/dockerfile:1

FROM golang:1.24.2-alpine AS build
WORKDIR /app

COPY src/go.mod ./
COPY src/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ratelimiter ./cmd/ratelimiter

FROM scratch
COPY --from=build /out/ratelimiter /ratelimiter
USER 65532:65532
EXPOSE 3000
ENTRYPOINT ["/ratelimiter"]
