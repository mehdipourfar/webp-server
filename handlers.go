package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/teris-io/shortid"
	"github.com/valyala/fasthttp"
	"regexp"
)

type Handler struct {
	Config             *Config
	CacheControlHeader []byte
	TaskMan            *TaskMan
}

var (
	CT_JPEG = "image/jpeg"
	CT_PNG  = "image/png"
	CT_WEBP = "image/webp"
	CT_JSON = "application/json"

	PATH_HEALTH = []byte("/health/")
	PATH_UPLOAD = []byte("/upload/")
	PATH_IMAGE  = []byte("/image/")
	PATH_DELETE = []byte("/delete/")

	CACHE_CONTROL = []byte("Cache-Control")

	ERROR_METHOD_NOT_ALLOWED     = []byte(`{"error": "Method not allowed"}`)
	ERROR_IMAGE_NOT_PROVIDED     = []byte(`{"error": "image_file field not provided"}`)
	ERROR_FILE_IS_NOT_IMAGE      = []byte(`{"error": "Provided file is not an accepted image"}`)
	ERROR_INVALID_TOKEN          = []byte(`{"error": "Invalid Token"}`)
	ERROR_IMAGE_NOT_FOUND        = []byte(`{"error": "Image not found"}`)
	ERROR_ADDRESS_NOT_FOUND      = []byte(`{"error": "Address not found"}`)
	ERROR_SERVER                 = []byte(`{"error": "Internal Server Error"}`)
	ERROR_TOO_BIG_REQUEST_HEADER = []byte(`{"error": "Too big request header"}`)
	ERROR_REQUEST_TIMEOUT        = []byte(`{"error": "Request timeout"}`)
	ERROR_CANNOT_PARSE_REQUEST   = []byte(`{"error": "Error when parsing request"}`)

	IMAGE_URI_REGEX  = regexp.MustCompile("/image/((?P<options>[0-9a-z,=-]+)/)?(?P<imageId>[0-9a-zA-Z_-]{9,12})$")
	DELETE_URI_REGEX = regexp.MustCompile("/delete/(?P<imageId>[0-9a-zA-Z_-]{9,12})$")

	// This variable makes us be able to mock ConvertAndCache function in tests

	ConvertFunction = Convert
)

func jsonResponse(ctx *fasthttp.RequestCtx, status int, body []byte) {
	ctx.SetStatusCode(status)
	ctx.SetContentType(CT_JSON)
	if body != nil {
		ctx.SetBody(body)
	}
}

// In case of ocurring any panic in code, this function will serve
// 500 error and log the error message.
func handlePanic(ctx *fasthttp.RequestCtx) {
	if err := recover(); err != nil {
		ctx.ResetBody()
		jsonResponse(ctx, 500, ERROR_SERVER)
		log.Println(err)
	}
}

// router function
func (handler *Handler) handleRequests(ctx *fasthttp.RequestCtx) {
	defer handlePanic(ctx)

	path := ctx.Path()

	if bytes.HasPrefix(path, PATH_IMAGE) {
		handler.handleFetch(ctx)
	} else if bytes.Equal(path, PATH_UPLOAD) {
		handler.handleUpload(ctx)
	} else if bytes.HasPrefix(path, PATH_DELETE) {
		handler.handleDelete(ctx)
	} else if bytes.Equal(path, PATH_HEALTH) {
		jsonResponse(ctx, 200, []byte(`{"status": "ok"}`))
	} else {
		jsonResponse(ctx, 404, ERROR_ADDRESS_NOT_FOUND)
	}
}

func (handler *Handler) tokenIsValid(ctx *fasthttp.RequestCtx) bool {
	return len(handler.Config.Token) == 0 ||
		handler.Config.Token == string(ctx.Request.Header.Peek("Token"))
}

func (handler *Handler) handleUpload(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		jsonResponse(ctx, 405, ERROR_METHOD_NOT_ALLOWED)
		return
	}

	if !handler.tokenIsValid(ctx) {
		jsonResponse(ctx, 401, ERROR_INVALID_TOKEN)
		return
	}

	fileHeader, err := ctx.FormFile("image_file")
	if err != nil {
		jsonResponse(ctx, 400, ERROR_IMAGE_NOT_PROVIDED)
		return
	}
	if imageValidated := ValidateImage(fileHeader); !imageValidated {
		jsonResponse(ctx, 400, ERROR_FILE_IS_NOT_IMAGE)
		return
	}

	imageId := shortid.GetDefault().MustGenerate()
	imagePath := ImageIdToFilePath(handler.Config.DataDir, imageId)
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		panic(err)
	}
	if err := fasthttp.SaveMultipartFile(fileHeader, imagePath); err != nil {
		panic(err)
	}
	jsonResponse(ctx, 200, []byte(fmt.Sprintf(`{"image_id": "%s"}`, imageId)))
}

func (handler *Handler) handleDelete(ctx *fasthttp.RequestCtx) {
	if !ctx.IsDelete() {
		jsonResponse(ctx, 405, ERROR_METHOD_NOT_ALLOWED)
		return
	}

	if !handler.tokenIsValid(ctx) {
		jsonResponse(ctx, 401, ERROR_INVALID_TOKEN)
		return
	}

	match := DELETE_URI_REGEX.FindSubmatch(ctx.Path())
	if len(match) != 2 {
		jsonResponse(ctx, 404, ERROR_ADDRESS_NOT_FOUND)
		return
	}
	imageId := string(match[1])
	imagePath := ImageIdToFilePath(handler.Config.DataDir, imageId)

	err := os.Remove(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			jsonResponse(ctx, 404, ERROR_IMAGE_NOT_FOUND)
			return
		}
		panic(err)
	}
	jsonResponse(ctx, 204, nil)
	return
}

func (handler *Handler) handleFetch(ctx *fasthttp.RequestCtx) {
	if !ctx.IsGet() {
		jsonResponse(ctx, 405, ERROR_METHOD_NOT_ALLOWED)
		return
	}
	options, imageId := parseImageUri(ctx.Path())
	if len(imageId) == 0 {
		jsonResponse(ctx, 404, ERROR_ADDRESS_NOT_FOUND)
		return
	}

	if len(options) == 0 {
		// user wants original file
		imagePath := ImageIdToFilePath(handler.Config.DataDir, imageId)
		if ok := handler.serveFileFromDisk(ctx, imagePath, true); !ok {
			jsonResponse(ctx, 404, ERROR_IMAGE_NOT_FOUND)
		}
		return
	}

	webpAccepted := bytes.Contains(ctx.Request.Header.Peek("accept"), []byte("webp"))

	imageParams, err := CreateImageParams(
		imageId,
		options,
		webpAccepted,
		handler.Config,
	)

	if err != nil {
		errorBody := []byte(fmt.Sprintf(`{"error": "Invalid options: %v"}`, err))
		jsonResponse(ctx, 400, errorBody)
		return
	}

	if webpAccepted {
		ctx.SetContentType(CT_WEBP)
	} else {
		ctx.SetContentType(CT_JPEG)
	}

	cacheFilePath := imageParams.GetCachePath(handler.Config.DataDir)
	if ok := handler.serveFileFromDisk(ctx, cacheFilePath, true); ok {
		// request served from cache
		return
	}
	// cache didn't exist

	if err := ValidateImageParams(imageParams, handler.Config); err != nil {
		errorBody := []byte(fmt.Sprintf(`{"error": "%v"}`, err))
		jsonResponse(ctx, 400, errorBody)
		return
	}

	imagePath := ImageIdToFilePath(handler.Config.DataDir, imageParams.ImageId)

	err = handler.TaskMan.Run(imageParams.GetMd5(), func() error {
		return ConvertFunction(imagePath, cacheFilePath, imageParams)
	})

	if err != nil {
		if os.IsNotExist(err) {
			jsonResponse(ctx, 404, ERROR_IMAGE_NOT_FOUND)
			return
		}
		panic(err)
	}

	ctx.Response.SetStatusCode(200)
	handler.serveFileFromDisk(ctx, cacheFilePath, false)
}

func (handler *Handler) handleErrors(ctx *fasthttp.RequestCtx, err error) {
	if _, ok := err.(*fasthttp.ErrSmallBuffer); ok {
		jsonResponse(ctx, fasthttp.StatusRequestHeaderFieldsTooLarge, ERROR_TOO_BIG_REQUEST_HEADER)
		ctx.Error("Too big request header", fasthttp.StatusRequestHeaderFieldsTooLarge)
	} else if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
		jsonResponse(ctx, fasthttp.StatusRequestTimeout, ERROR_REQUEST_TIMEOUT)
	} else {
		jsonResponse(ctx, fasthttp.StatusBadRequest, ERROR_CANNOT_PARSE_REQUEST)
	}
}

func (handler *Handler) serveFileFromDisk(ctx *fasthttp.RequestCtx, filePath string, checkExists bool) bool {
	if checkExists {
		info, err := os.Stat(filePath)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Println(err)
			}
			return false
		}
		if info.IsDir() {
			return false
		}
	}
	fasthttp.ServeFileUncompressed(ctx, filePath)
	status := ctx.Response.StatusCode()

	/* Instead of returning an error, fasthttp.ServeFile will reflect
	occurring of an error in status code and response body.
	We should detect if any error occured by checking the status code and
	then if any error occured, we will reset the response body
	to write our nice and pretty error message later. */
	ok := status < 400
	if !ok {
		ctx.Response.ResetBody()
	} else {
		ctx.Response.Header.SetBytesKV(CACHE_CONTROL, handler.CacheControlHeader)
	}

	return ok
}

func parseImageUri(requestPath []byte) (options, imageId string) {
	// options are in the format below:
	// w=200,h=200,fit=cover,quality=90

	match := IMAGE_URI_REGEX.FindStringSubmatch(string(requestPath))
	if len(match) != 4 {
		return
	}
	options, imageId = match[2], match[3]
	return
}
