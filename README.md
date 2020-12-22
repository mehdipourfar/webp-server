# webp-server (UNDER DEVELOPMENT)
A dynamic image resizer and format convertor server built on top of
[bimg](https://github.com/h2non/bimg) and [fasthttp](https://github.com/valyala/fasthttp)

### What is webp-server?
webp-server is a dynamic image resizer and format convertor server. Backend developers need to run this server on their vps machine and send image files from application server to it. It will return an `image_id` which needs to be saved on db.
By using that `image_id`, web clients can request images from webp-server and get them in appropriate size and format.

Here is an example request url for an image cropped to 500x500 size.

```code
https://example.com/image/w=500,h=500,fit=cover/(image_id)
```

### What are the benfits of webp format?
According to Google Developers website:
>  WebP is a modern image format that provides superior lossless and lossy compression for images on the web. Using WebP, webmasters and web developers can create smaller, richer images that make the web faster.

Although nowadays most browsers support WebP, lesser than 1% of websites
serve their images in this format. That's maybe because converting images to webp
can be complicated and time consuming or developers are not sure if 100% of their
users` browsers will support this format.

### How should client application check if the browser supports WebP?
There is no need to do anything. When browsers request for an image, they will send an `accept` header containing supported image formats. webp-server will lookup that header to see if the requesting browser supports webp format or not. If not, it will send the image in jpeg format.

### Isn't it resource expensive to convert images on each requests?
Yes, it is. For this reason, webp-server will cache each converted image after the first request.

### What about security topics such as DOS attack and heavily storage usage?
They are up to you. You can limit combinations of widths and heights or qualities that you will accept from the client in webp-server configuration. In case of serving requests from the cache, powered by `fasthttp`, webp-server is blazingly fast.

### Can web clients upload images to webp-server and send the `image_id` to web server?
It is strongly recommended not to do this and also not share your webp-server token
with frontend application for security reasons.
Frontend should upload image to backend, backend should upload it to wepb-server and store returning `image_id` in database.


## Installation
[bimg](https://github.com/h2non/bimg) is a golang program which communicates with libvips through C bindings. Since webp-server
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
There is an example configuration file named `example-config.yml` in code directory. Here is the list of parameters that you can configure:

* data_dir: Data directory in which images and cached images are
stored. Note that in this directory, there will be two separate directories
named `images` and `caches`. You can remove `caches` directory at any point
if you wanted to free up some disk space.

* server_address: Combination of ip:port. Default value is 127.0.0.1:8080
You can also set unix socket path for server address.

* token: The token that your backend application should send in request header for upload and delete operations.

* default_image_quality: When converting images, webp-server uses this value for conversion quality in case user omits quality option in request. Default value is 95. By decreasing this value, size and quality of the image will be decreased.

* valid_image_qualities: List of integer values from 50 to 100 which will be
accepted from users as quality option.
(Narrow down this values to prevent attackers from creating too many cache files for your images.)

* valid_image_sizes: List of string values in widthxheight format which will be accepted from users as width and height options. In case you want your users be able to set width=500 without providing height, you can add 500x0 in values list.
(Narrow down this values to prevent attackers from creating too many cache files for your images.)

* max_uploaded_image_size: Maximum size of accepted uploaded images in Megabytes.
