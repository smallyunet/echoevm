FROM golang:1.25.12-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION=devel
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o /out/echoevm ./cmd/echoevm

FROM alpine:3.24.1

RUN apk add --no-cache ca-certificates
COPY --from=build /out/echoevm /usr/local/bin/echoevm

USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/echoevm"]
CMD ["diff", "--web", "--addr", ":8080"]
