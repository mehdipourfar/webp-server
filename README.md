# webp-server
A dynamic image resizer and format convertor server built on top of
[bimg](https://github.com/h2non/bimg) and [fasthttp](https://github.com/valyala/fasthttp)


## FAQ

### What is webp-server?
webp-server is a dynamic image resizer and format convertor server.
Backend developers need to run this server on their vps machine and
send image files from application server to it. It will return an
`image_id` which needs to be saved on db.
By using that `image_id`, web clients can request images from webp-serevr
and get them in appropriate size and format.

Here is an example request url for an image cropped to 500x500 size.

```code
http://example.com/image/w=500,h=500/(image_id)
```

### What is the benfit of webp format?
According to Google Developers website:
>  WebP is a modern image format that provides superior lossless and
>  lossy compression for images on the web. Using WebP, webmasters
>  and web developers can create smaller, richer images that make the web faster.

Although nowadays most browsers support WebP, lesser than 1% of websites
serve their images in this format. That's maybe because converting images to webp
can be complicated and time consuming or developers are not sure if 100% of their
users` browsers will support this format.

### How should client application check if the browser supports webp?
There is no need to do anything. When browsers request for an image, they will send
an `accept` header containing supported image formats. webp-server will lookup
that header to see if the requesting browser supports webp format.
If not, it will send the image in jpeg.

### Isn't it resource expensive to convert images on each requests?
Yes, it is. For this reason, webp-server will cache each converted image
after first request.

### What about security topics such as DOS attack and heavily storage usage?
It is up to you. You can limit combinations of widths and heights that
you will accept from the client in webp-server config.

### Can web client upload images to webp server and send the `image_id` to web server?
It is strongly recommended not to do this and also not share your webp-server token
with frontend application for security reasons.


## Installation

```code
sudo apt install libvips-dev
```
