FROM public.ecr.aws/docker/library/golang:1.26.2-alpine3.23 AS build

ARG VERSION=debug

WORKDIR /app

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/cmd/ ./cmd/
COPY src/pkg/ ./pkg/

RUN CGO_ENABLED=0 go build -o /go/bin/vmGoat -ldflags="-X main.Version=$VERSION" ./cmd/vmGoat/main.go

# https://hub.docker.com/r/alpine/ansible
FROM registry.hub.docker.com/alpine/ansible:2.20.0@sha256:acea0c86ef95690d930b45737f988ba170055182f688b771e86f38fd9624605d AS production

RUN apk add --no-cache \
  py3-passlib

COPY base /mnt/base
COPY --chown=root:root --chmod=050 --from=build /go/bin/vmGoat /bin/vmGoat
COPY scenarios /mnt/scenarios

CMD ["/bin/vmGoat"]
