# Stage 1: Build Next.js static export
FROM docker.io/node:22-alpine AS frontend

WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

COPY web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM docker.io/golang:1.23-alpine AS backend

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=frontend /app/web/out ./web/out

RUN CGO_ENABLED=0 go build -o /kanba ./cmd/kanba

# Stage 3: Runtime
FROM docker.io/alpine:3.21

RUN apk add --no-cache ca-certificates

RUN addgroup -S kanba && adduser -S kanba -G kanba

WORKDIR /app

COPY --from=backend /kanba /app/kanba
COPY --from=backend /app/web/out ./web/out

RUN mkdir -p /app/data && chown kanba:kanba /app/data

EXPOSE 8080

ENV PORT=8080
ENV HOST=0.0.0.0

USER kanba

ENTRYPOINT ["/app/kanba"]
