package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/teris-io/shortid"
	"github.com/valyala/fasthttp"
	bimg "gopkg.in/h2non/bimg.v1"
	"regexp"
)

type Handler struct {
	Config *Config
}

var (
	CT_JPEG = "image/jpeg"
	CT_PNG  = "image/png"
	CT_WEBP = "image/webp"
	CT_GIF  = "image/gif"
	CT_JSON = "application/json"

	PATH_HEALTH = []byte("/health/")
	PATH_UPLOAD = []byte("/upload/")
	PATH_IMAGE  = []byte("/image/")
	PATH_DELETE = []byte("/delete/")

	ERROR_METHOD_NOT_ALLOWED = []byte(`{"error": "Method not allowed"}`)
	ERROR_IMAGE_NOT_PROVIDED = []byte(`{"error": "image_file field not provided"}`)
	ERROR_FILE_IS_NOT_IMAGE  = []byte(`{"error": "Provided file is not an accepted image"}`)
	ERROR_INVALID_TOKEN      = []byte(`{"error": "Invalid Token"}`)
	ERROR_INVALID_IMAGE_SIZE = []byte(`{"error": "Invalid image size"}`)
	ERROR_IMAGE_NOT_FOUND    = []byte(`{"error": "Image not found"}`)
	ERROR_ADDRESS_NOT_FOUND  = []byte(`{"error": "Address not found"}`)
	ERROR_SERVER             = []byte(`{"error": "Internal Server Error"}`)

	IMAGE_URI_REGEX  = regexp.MustCompile("/image/((?P<options>[0-9a-z,=-]+)/)?(?P<imageId>[0-9a-zA-Z_-]{9,12})$")
	DELETE_URI_REGEX = regexp.MustCompile("/delete/(?P<imageId>[0-9a-zA-Z_-]{9,12})$")
)

func jsonResponse(ctx *fasthttp.RequestCtx, status int, body []byte) {
	ctx.SetStatusCode(status)
	ctx.SetContentType(CT_JSON)
	if body != nil {
		ctx.SetBody(body)
	}
}

func serveFileFromDisk(ctx *fasthttp.RequestCtx, filePath string, checkExists bool) bool {
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
	ok := status < 400
	if !ok {
		ctx.Response.ResetBody()
	}

	fi, _ := os.Stat(filePath)
	stat := fi.Sys().(*syscall.Stat_t)
	atime := time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	fmt.Println(filePath, atime)
	return ok
}

func parseImageUri(requestPath []byte) (options, imageId string) {
	match := IMAGE_URI_REGEX.FindStringSubmatch(string(requestPath))
	if len(match) != 4 {
		return
	}
	options, imageId = match[2], match[3]
	return
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
	if len(handler.Config.TOKEN) != 0 &&
		handler.Config.TOKEN != string(ctx.Request.Header.Peek("Token")) {
		return false
	}
	return true
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

	imageId := shortid.GetDefault().MustGenerate()
	fileHeader, err := ctx.FormFile("image_file")
	if err != nil {
		jsonResponse(ctx, 400, ERROR_IMAGE_NOT_PROVIDED)
		return
	}
	if imageValidated := ValidateImage(fileHeader); !imageValidated {
		jsonResponse(ctx, 400, ERROR_FILE_IS_NOT_IMAGE)
		return
	}

	parentDir, filePath := ImageIdToFilePath(handler.Config.DATA_DIR, imageId)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		panic(err)
		return
	}
	if err := fasthttp.SaveMultipartFile(fileHeader, filePath); err != nil {
		panic(err)
		return
	}
	jsonResponse(ctx, 200, []byte(fmt.Sprintf(`{"image_id": "%s"}`, imageId)))
}

func (handler *Handler) handleFetch(ctx *fasthttp.RequestCtx) {
	if !ctx.IsGet() {
		jsonResponse(ctx, 405, ERROR_METHOD_NOT_ALLOWED)
		return
	}
	options, imageId := parseImageUri(ctx.Path())
	if imageId == "" {
		jsonResponse(ctx, 404, ERROR_ADDRESS_NOT_FOUND)
		return
	}

	if options == "" {
		// serve original file
		_, path := ImageIdToFilePath(handler.Config.DATA_DIR, imageId)
		if ok := serveFileFromDisk(ctx, path, true); !ok {
			jsonResponse(ctx, 404, ERROR_IMAGE_NOT_FOUND)
		}
		return
	}

	imageParams, err := CreateImageParams(
		imageId,
		options,
		bytes.Contains(ctx.Request.Header.Peek("accept"), []byte("webp")),
		handler.Config,
	)

	if err != nil {
		errorBody := []byte(fmt.Sprintf(`{"error": "Invalid options: %v"}`, err))
		jsonResponse(ctx, 400, errorBody)
		return
	}

	cacheParentDir, cacheFilePath := imageParams.GetCachePath(handler.Config.DATA_DIR)
	if ok := serveFileFromDisk(ctx, cacheFilePath, true); ok {
		// request served from cache
		return
	}

	if !ValidateImageSize(imageParams.Width, imageParams.Height, handler.Config) {
		jsonResponse(ctx, 403, ERROR_INVALID_IMAGE_SIZE)
		return
	}
	_, imageFilePath := ImageIdToFilePath(handler.Config.DATA_DIR, imageParams.ImageId)

	imgBuffer, err := bimg.Read(imageFilePath)

	if err != nil {
		if os.IsNotExist(err) {
			jsonResponse(ctx, 404, ERROR_IMAGE_NOT_FOUND)
			return
		}
		panic(err)
	}

	convertedImage, _, err := Convert(imgBuffer, imageParams)
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(cacheParentDir, 0755); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(cacheFilePath, convertedImage, 0604); err != nil {
		panic(err)
	}

	ctx.Response.SetStatusCode(200)
	serveFileFromDisk(ctx, cacheFilePath, false)
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
	_, imagePath := ImageIdToFilePath(handler.Config.DATA_DIR, imageId)

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
