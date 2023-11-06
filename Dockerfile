FROM golang:1.20-bullseye as base

RUN adduser \
  --disabled-password \
  --gecos "" \
  --home "/nonexistent" \
  --shell "/sbin/nologin" \
  --no-create-home \
  --uid 65532 \
  small-user


WORKDIR /src/smallest-golang/app/
COPY . .
RUN ls -lah
RUN go mod download
RUN go mod verify
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go build -buildvcs=false -a -x -ldflags="-w -s" -o /main cmd/exporter/main.go

FROM scratch
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group
COPY --from=base /main .
USER small-user:small-user
CMD ["./main"]