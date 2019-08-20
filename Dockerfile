FROM golang:1.16.0-alpine3.13 as backend

WORKDIR /root

COPY [".", "."]

RUN go build -o=main ./go

# -------

FROM alpine:3.13 as runner

# Dependencies:
#   postgresql-client: for psql (DB update)

RUN apk add --no-cache postgresql-client

WORKDIR /root

COPY [".", "."]

COPY --from=backend /root/main .

CMD ["./misc/start"]
