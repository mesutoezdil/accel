FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /siltide .

# A minimal base rather than scratch: purego's dlopen needs a dynamic
# loader. The DaemonSet mounts the host's real one over /usr; gcompat
# covers a plain `docker run`, where NVML's glibc .so has none otherwise.
FROM alpine:3.20
RUN apk add --no-cache gcompat
COPY --from=build /siltide /usr/local/bin/siltide
ENTRYPOINT ["siltide"]
CMD ["--service", "--listen", ":9800"]
