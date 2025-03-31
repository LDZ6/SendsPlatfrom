FROM golang:alpine AS base

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /app
COPY . .


RUN go mod download
RUN go build -o ./cmd/school/main ./app/school/cmd/main.go


FROM alpine AS final
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk update && apk add tzdata
COPY --from=base /app/config/ /config/
COPY --from=base /app/cmd/school/main .
EXPOSE 10004
CMD ["./main"]