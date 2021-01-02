ARG GOLANG_VERSION=1.15.6
FROM golang:${GOLANG_VERSION} as builder

ARG WEBP_SERVER_VERSION=1.0.0
ARG LIBVIPS_VERSION=8.10.5

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
        apt-get install --no-install-recommends -y \
        file ca-certificates automake build-essential curl fftw3-dev \
        liborc-0.4-dev libexif-dev libglib2.0-dev libexpat1-dev \
        libpng-dev libjpeg62-turbo-dev libwebp-dev

RUN cd /tmp && \
        curl -fsSLO https://github.com/libvips/libvips/releases/download/v${LIBVIPS_VERSION}/vips-${LIBVIPS_VERSION}.tar.gz && \
        tar zvxf vips-${LIBVIPS_VERSION}.tar.gz && \
        cd /tmp/vips-${LIBVIPS_VERSION} && \
            CFLAGS="-g -O3" CXXFLAGS="-D_GLIBCXX_USE_CXX11_ABI=0 -g -O3" \
            ./configure \
            --disable-debug \
            --disable-dependency-tracking \
            --disable-introspection \
            --disable-static \
            --without-tiff \
            --enable-gtk-doc-html=no \
            --enable-gtk-doc=no && \
        make && \
        make install && \
        ldconfig

WORKDIR ${GOPATH}/src/github.com/mehdipourfar/webp-server
ENV GO111MODULE=on
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go test -race
RUN go build -a -o ${GOPATH}/bin/webp-server github.com/mehdipourfar/webp-server


FROM debian:buster-slim

ARG WEBP_SERVER_VERSION

COPY --from=builder /usr/local/lib /usr/local/lib
COPY --from=builder /go/bin/webp-server /usr/local/bin/webp-server
COPY ./docker-entrypoint.sh /docker-entrypoint.sh

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
        apt-get install --no-install-recommends -y \
        libexpat1 libglib2.0-0 libexif12 libjpeg62-turbo libpng16-16 \
        libwebp6 libwebpmux3 libwebpdemux2 fftw3 liborc-0.4-0 curl && \
        apt-get autoremove -y && \
        apt-get autoclean && \
        apt-get clean && \
        rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
        groupadd -r -g 999 webp-server && useradd -r -g webp-server -u 999 --home-dir=/var/lib/webp-server webp-server; \
        mkdir -p /var/lib/webp-server; \
        chown webp-server:webp-server /var/lib/webp-server;

ARG BUILD_DATE
LABEL maintainer="mehdipourfar@gmail.com" \
        org.label-schema.schema-version="1.0" \
        org.label-schema.name="webp-server" \
        org.label-schema.build-date=$BUILD_DATE \
        org.label-schema.description="Simple and minimal image server capable of storing, resizing, converting and caching images." \
        org.label-schema.url="https://github.com/mehdipourfar/webp-server" \
        org.label-schema.vcs-url="https://github.com/mehdipourfar/webp-server" \
        org.label-schema.version="${WEBP_SERVER_VERSION}" \
        org.label-schema.docker.cmd="docker run -d -v webp_server_volume:/var/lib/webp-server --name webp-server -e TOKEN='MY_STRONG_TOKEN' -p 8080:8080 webp-server"


VOLUME /var/lib/webp-server
COPY ./docker-config.yml /var/lib/webp-server/config.yml
ENTRYPOINT ["./docker-entrypoint.sh"]
HEALTHCHECK CMD curl --fail http://localhost:8080/health/ || exit 1
USER webp-server
EXPOSE 8080
