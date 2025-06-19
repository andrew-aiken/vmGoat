FROM public.ecr.aws/docker/library/golang:1.24.4-alpine3.22 AS build

ARG VERSION=debug

WORKDIR /app

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/cmd/ ./cmd/
COPY src/pkg/ ./pkg/

RUN CGO_ENABLED=0 go build -o /go/bin/vmGoat -ldflags="-X main.Version=$VERSION" ./cmd/vmGoat/main.go

# https://hub.docker.com/r/alpine/ansible
FROM registry.hub.docker.com/alpine/ansible:2.18.6@sha256:79af548c9c5f23e3eed286389c309b228d0787f8b0b375576eb0d20c4d80efa6 AS production

RUN apk add --no-cache \
  py3-passlib

COPY base /mnt/base
COPY --chown=root:root --chmod=010 --from=build /go/bin/vmGoat /
COPY scenarios /mnt/scenarios

CMD ["/vmGoat"]
