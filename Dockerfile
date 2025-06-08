FROM public.ecr.aws/docker/library/golang:1.24.3-alpine3.21 AS build

WORKDIR /app

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/cmd/ ./cmd/
COPY src/pkg/ ./pkg/

RUN CGO_ENABLED=0 go build -o /go/bin/vmGoat ./cmd/vmGoat/main.go

# https://hub.docker.com/r/alpine/ansible
FROM alpine/ansible:2.18.6@sha256:81b0fac0c7a9a1b71a0ee4e58a4754c6e1ba933993b0dcac7bc50e54b4985626 AS production

RUN apk add --no-cache \
  py3-passlib

COPY base /mnt/base
COPY --chown=root:root --chmod=010 --from=build /go/bin/vmGoat /
COPY scenarios /mnt/scenarios

CMD ["/vmGoat"]
