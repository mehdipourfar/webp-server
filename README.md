# WebpServe
WebpServe is an image resize and coversion server.
You run the server and send your image files from your application server to it.
Then it returns you an image id and you store it in your database. After that,
you just need to send your image_id to your client application and it is it's
responsibility to create the image url with appropriate options.

Here is an example request url for an image cropped to 500x500 size.

```code
http://example.com/image/w=500,h=500,fit=cover,format=auto/(image_id)
```
