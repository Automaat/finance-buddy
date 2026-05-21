# Build
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/proxy .

# Run
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/proxy /usr/local/bin/proxy
COPY routes.yaml /etc/proxy/routes.yaml
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/proxy", "--config", "/etc/proxy/routes.yaml", "--addr", ":8080"]
