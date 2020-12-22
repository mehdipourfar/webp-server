# webp-server (UNDER DEVELOPMENT)
Simple and minimal image server capable of storing, resizing, converting and caching images.

![Flowchart](https://github.com/mehdipourfar/webp-server/raw/master/docs/flowchart.jpg)


## FAQ
* ### What is webp-server?
  `webp-server` is a dynamic image resizer and format convertor server built on top of [bimg](https://github.com/h2non/bimg) and [fasthttp](https://github.com/valyala/fasthttp). Backend developers can run this program on their server machines and upload images to it instead of storing them. It will return an `image_id` which needs to be saved on database by backend application (on a `varchar` field with length of at least 12).
  By using that `image_id`, web clients can request images from `webp-server` and get them in the appropriate size and format.

    Here is an example request url for an image cropped to 500x500 size.

    ```code
    https://example.com/image/w=500,h=500,fit=cover/(image_id)
    ```

* ### What are the benfits of WebP format?
  According to Google Developers website:
  >  WebP is a modern image format that provides superior lossless and lossy compression for images on the web. Using WebP, webmasters and web developers can create smaller, richer images that make the web faster.

    Although nowadays most browsers support WebP, lesser than 1% of websites serve their images in this format. That's maybe because converting images to WebP can be complicated and time consuming or developers are not sure if all browsers will support this format.

* ### How should client application check if the browser supports WebP?
    There is no need to do anything. When browsers request for an image, they will send an accept header containing supported image formats. `webp-server` will lookup that header to see if the requesting browser supports webp format or not. If not, it will send the image in jpeg format.
* ### Isn't it resource expensive to convert images on each requests?
  Yes, it is. For this reason, webp-server will cache each converted image after the first request.

* ### What about security topics such as DOS attack or heavily storage space usage?
  They are up to you. You can limit combinations of widths and heights or qualities that you will accept from the client in webp-server configuration and by doing that you will narrow down the type of accepted requests for generating images. In case of serving requests from the cache, powered by `fasthttp`, `webp-server` can be blazingly fast.

* ### Can web clients upload images to webp-server and send the `image_id` to web server?
  It is strongly recommended not to do this and also not share your webp-server token with frontend application for security reasons. Frontend should upload image to backend, backend should upload it to wepb-server and store the returning `image_id` in database.

* ### What is the advantage of using `webp-server` instead of similar projects?
  It is simple and minimal and has been created to work along the backend applications for serving images of websites in webp format.

## Installation
[bimg](https://github.com/h2non/bimg) is a golang program which communicates with libvips through C bindings. Since `webp-server`
uses `bimg` for image conversion, you need to install `libvips-dev` as
a dependency.


```sh
sudo apt install libvips-dev
go get -u github.com/mehdipourfar/webp-server
```

## Running
```sh
webp-server -config /path/to/config.yml
```

## Configuration
There is an example configuration file [example-config.yml](https://github.com/mehdipourfar/webp-server/blob/master/example-config.yml) in code directory. Here is the list of parameters that you can configure:

* `data_dir`: Data directory in which images and cached images are
stored. Note that in this directory, there will be two separate directories
named `images` and `caches`. You can remove `caches` directory at any point
if you wanted to free up some disk space.

* `server_address`: Combination of ip:port. Default value is 127.0.0.1:8080
You can also set unix socket path for server address (unix:/path/to/socket.sock)

* `token`: The token that your backend application should send in request header for upload and delete operations.

* `default_image_quality`: When converting images, webp-server uses this value for conversion quality in case user omits quality option in request. Default value is 95. By decreasing this value, size and quality of the image will be decreased.

* `valid_image_qualities`: List of integer values from 50 to 100 which will be
accepted from users as quality option.
(Narrow down this values to prevent attackers from creating too many cache files for your images.)

* `valid_image_sizes`: List of string values in (width)x(height) format which will be accepted from users as width and height options. In case you want your users be able to set width=500 without providing height, you can add 500x0 in values list.
(Narrow down this values to prevent attackers from creating too many cache files for your images.)

* `max_uploaded_image_size`: Maximum size of accepted uploaded images in Megabytes.


## Backend APIs
* `/upload/  [Method: POST]`: Accepts image in multipart/form-data format with field name of `image_file`. You should also pass the `Token` previously set in your configuration file as header. All responses are in JSON format. If request is successful, you will get `200` status code with such body: `{"image_id": "lulRDHbMg"}` (Note that `image_id` length can vary from 9 to 12). Otherwise, depending on the error, you will get `4xx` for `5xx` status code with body like this: `{"error": "reason of error"}`.

    Example:
    ```sh
    curl -H 'Token: 456e910f-3d07-470d-a862-1deb1494a38e' -X POST -F 'image_file=@/path/to/image.png' http://127.0.0.1:8080/upload/
    ```

* `/delete/(image_id)  [Method: DELETE]`: Accepts `image_id` as url parameter. If the image is deleted without a problem, server will return `204` status code with an empty body. Otherwise, it will return `4xx` or `5xx` with error message in JSON format.

    Example:
    ```sh
    curl -H 'Token: 456e910f-3d07-470d-a862-1deb1494a38e' -X DELETE "http://localhost:8080/delete/lulRDHbMg";
    ```

* `/health/  [Method: GET]`: It returns `200` status code if server is up and running. It can be used by container managers to check the status of a container.


## Frontend APIs
* `/image/(image_id)  [Method: GET]`: Returns the image which has been uploaded to webp-server in original size and format.

* `/image/(filter_options)/(image_id)  [Method: GET]`: Returns the filtered image with content type based on `accept` header of the browser. Filter options can be these parameters:
  * `w`, `width`: Width of requested image.
  * `h`, `height`: height of requested image.
  * `q`, `quality`: quality of requested image.
  x* `fit`: Accepts `cover`, `contain` and `scale-down` as value.
    * `cover`: Image will be resized to exactly fill the entire area specified by `width` and `height`, and will cropped if necessary.
    * `contain`: Image will be resized (shrunk or enlarged) to be as large as possible within the given `width` or `height` while preserving the aspect ratio.
    * `scale-down`: Image will be shrunk in size to fully fit within the given `width` or `height`, but won’t be enlarged.

Some example image urls:
```
http://example.com/image/w=500,h=500/lulRDHbMg
http://example.com/image/w=500,h=500,q=95/lulRDHbMg
http://example.com/image/w=500,h=500,fit=cover/lulRDHbMg
http://example.com/image/w=500,h=500,fit=contain/lulRDHbMg
http://example.com/image/w=500,fit=contain/lulRDHbMg
```

## Reverse Proxy

`webp-server` does not support SSL or domain name validation. It is recommended to use a reverse proxy such as [nginx](https://www.nginx.com/) in front of it. It should only cover frontend APIs. Backend APIs should be called locally. Here is a minimal `nginx` configuration that redirects all the paths which starts with /image/ to webp-server.

``` nginx

upstream webp_server {
    server 127.0.0.1:8080 fail_timeout=0;
}

server {
   ##
   ##

   location /image/ {
        proxy_redirect off;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Scheme $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Protocol $scheme
        proxy_pass http://webp_server;
    }
}

```
