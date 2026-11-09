FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
# Genera go.sum si aun no existe (luego puedes versionarlo con `go mod tidy`).
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
