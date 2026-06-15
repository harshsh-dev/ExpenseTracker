# --- build ---
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

# --- run ---
FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/server /app/server
# snapshot persists here; mount a volume to survive redeploys
ENV DATA_PATH=/app/data/snapshot.json
ENV PORT=8080
# enabled features: "all" or a comma-separated subset, e.g. "income,expenses"
# (override at runtime: docker run -e FEATURES=income,expenses ...)
ENV FEATURES=all
VOLUME /app/data
EXPOSE 8080
CMD ["/app/server"]
