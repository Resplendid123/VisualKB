# --- builder ---
FROM golang:alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

COPY . .

# alpine musl 须关 CGO。
ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /out/server ./cmd/server

# --- runtime ---
FROM alpine:latest

RUN addgroup -S kb && adduser -S kb -G kb
WORKDIR /app
COPY --from=builder /out/server /app/server
COPY configs /app/configs
COPY internal/infra/ai/skills /app/internal/infra/ai/skills
RUN chown -R kb:kb /app

USER kb
EXPOSE 8889
ENTRYPOINT ["/app/server"]