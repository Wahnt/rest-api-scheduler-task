FROM golang:1.26.3-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download


COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o /out/app .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/app /app
COPY --from=builder /src/web /web

ENV STATIC_DIR=/web
ENV TODO_PORT=7540

EXPOSE 7540
USER nonroot:nonroot
ENTRYPOINT ["/app"]
