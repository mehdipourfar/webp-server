# webp-server
Simple and minimal image server capable of storing, resizing, converting, and caching images. You can quickly find out how it works by looking at the flowchart below.

[![codecov](https://codecov.io/gh/mehdipourfar/webp-server/branch/master/graph/badge.svg?token=CYJANFYQIM)](https://codecov.io/gh/mehdipourfar/webp-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/mehdipourfar/webp-server)](https://goreportcard.com/report/github.com/mehdipourfar/webp-server)
[![Release](https://img.shields.io/github/v/release/mehdipourfar/webp-server?sort=semver)](https://github.com/mehdipourfar/webp-server/releases)

<p align="center">
  <img src="https://github.com/mehdipourfar/webp-server/raw/master/docs/flowchart.jpg" alt="Flowchart"/>
</p>


## Contents

- [Quickstart](#quickstart)
- [FAQ](#faq)
- [Installation](#installation)
  - [Docker](#docker)
  - [Download Binary](#download-binary)
  - [Build From Source](#build-from-source)
- [Configuration](#configuration)
- [Backend APIs](#backend-apis)
- [Frontend APIs](#frontend-apis)
- [Reverse Proxy](#reverse-proxy)


## Quickstart
Run a docker container of `webp-server`.
```sh
docker run -d -v webp_server_volume:/var/lib/webp-server --name webp-server -e TOKEN='MY_STRONG_TOKEN' -p 127.0.0.1:8080:8080 ms68/webp-server
```
Upload an image:

``` sh
curl -H 'Token: MY_STRONG_TOKEN' -X POST -F 'image_file=@/path/to/image.jpg' http://127.0.0.1:8080/upload/

# this api will return an image_id
```

Open these urls in your browser.

```
http://127.0.0.1:8080/image/width=500,height=500,fit=contain,quality=100/{image_id}
http://127.0.0.1:8080/image/width=300,height=300,fit=cover,quality=90/{image_id}
```

For supporting more image sizes and qualities, you should edit the config file which resides in `webp_server_volume`:

``` sh
docker volume ls -f name=webp_server_volume --format "{{ .Mountpoint }}"
```

And then, restart the server:

``` sh
docker container restart webp-server
```



## FAQ
* ### What is webp-server?
  `webp-server` is a dynamic image resizer and format converter server built on top of [libvips](https://github.com/libvips/libvips), [bimg](https://github.com/h2non/bimg), and [fasthttp](https://github.com/valyala/fasthttp). Backend developers can run this program on their server machines and upload images to it instead of storing them. It will return an `image_id` which needs to be saved on a database by the backend application (on a `varchar` field with a length of at least 12).
  By using that `image_id`, web clients can request images from `webp-server` and get them in the appropriate size and format.

    Here is an example request URL for an image cropped to 500x500 size.

    ```code
    https://example.com/image/w=500,h=500,fit=cover/(image_id)
    ```

* ### What are the benfits of serving images in WebP format?
  According to Google Developers website:
  >  WebP is a modern image format that provides superior lossless and lossy compression for images on the web. Using WebP, webmasters and web developers can create smaller, richer images that make the web faster.

    Although nowadays most web browsers support WebP, less than 1% of websites serve their images in this format. That's maybe because converting images to WebP can be complicated and time-consuming or developers don't know what to do with the browsers which don't support WebP.

* ### What can webp-server do about the browsers which don't support WebP?
    When browsers request an image, they will send an accept header containing supported image formats. `webp-server` will lookup that header to see if the browser supports WebP or not. If not, it will send the image in JPEG.

* ### Isn't it resource expensive to convert images on each request?
  Yes, it is. For this reason, `webp-server` will cache each converted image after the first request.

* ### What about security topics such as DOS attacks or heavy storage usage?
  It is up to you. You can limit the combinations of widths and heights or qualities that you accept from the client in `webp-server` configuration file, and by doing that you will narrow down the type of accepted requests for generating images. In case of serving requests from the cache, powered by `fasthttp`, `webp-server` can be blazingly fast.

* ### Can web clients upload images to `webp-server` and send the `image_id` to a web server?
  It is strongly recommended not to do this and also do not share your `webp-server` token with frontend applications for security reasons. Process should be like this: Frontend uploads the image to the backend, backend uploads it to wepb-server, and stores the returning `image_id` in database.

* ### What is the advantage of using `webp-server` instead of similar projects?
  It is simple and minimal and has been designed to work along with the backend applications for serving images of websites in WebP format. It does not support all kinds of manipulations that one can do with images. It does a few things and tries to do them perfectly.


## Installation
There are two methods for running `webp-server`. Either use docker or build it yourself:

### Docker

```sh
docker run -d -v webp_server_volume:/var/lib/webp-server --name webp-server -e TOKEN='MY_STRONG_TOKEN' -p 127.0.0.1:8080:8080 ms68/webp-server
```

### Download Binary

``` sh
sudo apt install libvips
wget https://github.com/mehdipourfar/webp-server/releases/download/v1.0.0/webp-server_1.0.0_linux_amd64.tar.gz
```

### Build From Source

``` sh
sudo apt install libvips-dev git


## in Case you don't have Golang installed on your system.

wget https://golang.org/dl/go1.15.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.15.6.linux-amd64.tar.gz
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin

go get -u -v github.com/mehdipourfar/webp-server
sudo cp $HOME/go/bin/webp-server /usr/bin/


# Download and edit `example-config.yml` to your desired config
wget https://raw.githubusercontent.com/mehdipourfar/webp-server/master/example-config.yml

# Run the server:
webp-server -config example-config.yml

```


## Configuration
There is an example configuration file [example-config.yml](https://github.com/mehdipourfar/webp-server/blob/master/example-config.yml) in the code directory. Here is the list of parameters that you can configure:

* `data_dir`: Data directory in which images and cached images are stored. Note that in this directory, there will be two separate directories named `images` and `caches`. You can remove the caches directory at any point in time if you wanted to free up some disk space.

* `server_address`: Combination of ip:port. Default value is 127.0.0.1:8080. You can also set unix socket path for server address (unix:/path/to/socket.sock)

* `token`: The token that your backend application should send in the request header for upload and delete operations.

* `default_image_quality`: When converting images, `webp-server` uses this value for conversion quality in case the user omits the quality option in the request. The default value is 95. By decreasing this value, size and quality of the image will be decreased.

* `valid_image_qualities`: List of integer values from 10 to 100 which will be
accepted from users as the quality option.
(Narrow down these values to prevent attackers from creating too many cache files for your images.)

* `valid_image_sizes`: List of string values in (width)x(height) format which will be accepted from users as width and height options. In case you want your users to be able to set width=500 without providing height, you can add 500x0 to the values list.
(Narrow down these values to prevent attackers from creating too many cache files for your images.)

* `max_uploaded_image_size`: Maximum size of accepted uploaded images in Megabytes.

* `debug`: When set to `true` `/image/` API does not check if width, height, and quality are included in `valid_image_sizes` and `valid_image_qualities`. It can be useful when you are developing your frontend applications and you are not yet sure which sizes and qualities you want. But do not set it to `true` on production server.


## Backend APIs
* `/upload/  [Method: POST]`: Accepts image in multipart/form-data format with a field name of `image_file`. You should also pass the `Token` previously set in your configuration file as a header. All responses are in JSON format. If request is successful, you will get `200` status code with such body: `{"image_id": "lulRDHbMg"}` (Note that `image_id` length can vary from 9 to 12). Otherwise, depending on the error, you will get `4xx` or `5xx` status code with a body like this: `{"error": "reason of error"}`.

    Example:
    ```sh
    curl -H 'Token: 456e910f-3d07-470d-a862-1deb1494a38e' -X POST -F 'image_file=@/path/to/image.png' http://127.0.0.1:8080/upload/
    ```

* `/delete/(image_id)  [Method: DELETE]`: Accepts `image_id` as URL parameter. If the image is deleted without a problem, the server will return `204` status code with an empty body. Otherwise, it will return `4xx` or `5xx` with an error message in JSON format.

    Example:
    ```sh
    curl -H 'Token: 456e910f-3d07-470d-a862-1deb1494a38e' -X DELETE "http://localhost:8080/delete/lulRDHbMg";
    ```

* `/health/  [Method: GET]`: It returns `200` status code if the server is up and running. It can be used by container managers to check the status of a `webp-server` container.


## Frontend APIs
* `/image/(image_id)  [Method: GET]`: Returns the image which has been uploaded to `webp-server` in original size and format.

* `/image/(filter_options)/(image_id)  [Method: GET]`: Returns the filtered image with content-type based on `Accept` header of the browser. Filter options can be these parameters:
  * `w`, `width`: Width of the requested image.
  * `h`, `height`: Height of the requested image.
  * `q`, `quality`: Quality of the requested image. The default value should be set in the server config.
  * `fit`: Accepts `cover`, `contain` and `scale-down` as value.
    * `contain`: Image will be resized (shrunk or enlarged) to be as large as possible within the given `width` or `height` while preserving the aspect ratio. This is the default value for fit.
    * `scale-down`: Image will be shrunk in size to fully fit within the given `width` or `height`, but won’t be enlarged.
    * `cover`: Image will be resized to exactly fill the entire area specified by `width` and `height`, and will cropped if necessary.

Some example image urls:
```
http://example.com/image/w=500,h=500/lulRDHbMg
http://example.com/image/w=500,h=500,q=95/lulRDHbMg
http://example.com/image/w=500,h=500,fit=cover/lulRDHbMg
http://example.com/image/w=500,h=500,fit=contain/lulRDHbMg
http://example.com/image/w=500,fit=contain/lulRDHbMg
```

## Reverse Proxy

`webp-server` does not support SSL or domain name validation. It is recommended to use a reverse proxy such as [nginx](https://www.nginx.com/) in front of it. It should only cover frontend APIs. Backend APIs should be called locally. Here is a minimal `nginx` configuration that redirects all the paths which start with `/image/` to `webp-server`.

``` nginx

upstream webp_server {
    server 127.0.0.1:8080 fail_timeout=0;
}

server {
   # ...

   location /image/ {
        proxy_redirect off;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Scheme $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Protocol $scheme;
        proxy_pass http://webp_server;
    }
}

```

## Security Checklist
* Set `debug` config to `false` value in production.
* Narrow down `valid_image_qualities` and `valid_image_sizes` to the values you really want.
* From the outside of the server, `webp-server` address should not be accessible, and users should only be able to see the `/image/` path through your reverse proxy.
