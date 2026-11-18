FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/ima-mcp-server ./cmd/ima-mcp-server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ima-mcp-server /ima-mcp-server
ENTRYPOINT ["/ima-mcp-server"]
