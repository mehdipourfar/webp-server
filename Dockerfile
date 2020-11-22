FROM alpine:3.10.2

RUN apk add --no-cache --virtual .build-deps gcc musl-dev openssl go g++ make

RUN apk add --update --no-cache libc6-compat curl
RUN apk add vips-dev fftw-dev build-base --no-cache \
        --repository https://dl-3.alpinelinux.org/alpine/edge/testing/ \
        --repository https://dl-3.alpinelinux.org/alpine/edge/main

WORKDIR /root/xerox
COPY . .
RUN GPATH=/root/go GOOS=linux GOARCH=amd64 go install

RUN apk del .build-deps
CMD ["/root/go/bin/xerox"]
