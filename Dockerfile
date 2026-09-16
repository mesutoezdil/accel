FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /accel .

# A minimal base rather than scratch: purego's dlopen needs the dynamic
# loader of the host, which the DaemonSet mounts from /usr.
FROM alpine:3.20
COPY --from=build /accel /usr/local/bin/accel
ENTRYPOINT ["accel"]
CMD ["--service", "--listen", ":9800"]
