package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/teris-io/shortid"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"regexp"
)

var (
	PathHealth = []byte("/health/")
	PathUpload = []byte("/upload/")
	PathImage  = []byte("/image/")
	PathDelete = []byte("/delete/")

	ImageRegex  = regexp.MustCompile("/image/((?P<options>[0-9a-z,=-]+)/)?(?P<imageID>[0-9a-zA-Z_-]{9,12})$")
	DeleteRegex = regexp.MustCompile("/delete/(?P<imageID>[0-9a-zA-Z_-]{9,12})$")

	CacheControlKey = []byte("Cache-Control")

	ErrorMethodNotAllowed = []byte(`{"error": "Method not allowed"}`)
	ErrorImageNotProvided = []byte(`{"error": "image_file field not provided"}`)
	ErrorFileIsNotImage   = []byte(`{"error": "Provided file is not an accepted image"}`)
	ErrorInvalidToken     = []byte(`{"error": "Invalid Token"}`)
	ErrorImageNotFound    = []byte(`{"error": "Image not found"}`)
	ErrorAddressNotFound  = []byte(`{"error": "Address not found"}`)
	ErrorServerError      = []byte(`{"error": "Internal Server Error"}`)

	// This variable makes us be able to mock convert function in tests
	convertFunction = convert
)

type Handler struct {
	Config             *Config
	CacheControlHeader []byte
	TaskManager        *TaskManager
}

func createServer(config *Config) *fasthttp.Server {
	handler := &Handler{Config: config}
	if config.HTTPCacheTTL == 0 {
		handler.CacheControlHeader = []byte("private, no-cache, no-store, must-revalidate")
	} else {
		handler.CacheControlHeader = []byte(fmt.Sprintf("max-age=%d", config.HTTPCacheTTL))
	}
	handler.TaskManager = NewTaskManager(config.ConvertConcurrency)
	return &fasthttp.Server{
		Handler:               handler.handleRequests,
		ErrorHandler:          handler.handleErrors,
		NoDefaultServerHeader: true,
		MaxRequestBodySize:    config.MaxUploadedImageSize * 1024 * 1024,
		ReadTimeout:           time.Duration(5 * time.Second),
	}
}

func jsonResponse(ctx *fasthttp.RequestCtx, status int, body []byte) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	if body != nil {
		ctx.SetBody(body)
	}
}

// In case of ocurring any panic in code, this function will serve
// 500 error and log the error message.
func handlePanic(ctx *fasthttp.RequestCtx) {
	if err := recover(); err != nil {
		ctx.ResetBody()
		jsonResponse(ctx, 500, ErrorServerError)
		log.Println(err)
	}
}

// router function
func (handler *Handler) handleRequests(ctx *fasthttp.RequestCtx) {
	defer handlePanic(ctx)

	path := ctx.Path()

	if bytes.HasPrefix(path, PathImage) {
		handler.handleFetch(ctx)
	} else if bytes.Equal(path, PathUpload) {
		handler.handleUpload(ctx)
	} else if bytes.HasPrefix(path, PathDelete) {
		handler.handleDelete(ctx)
	} else if bytes.Equal(path, PathHealth) {
		jsonResponse(ctx, 200, []byte(`{"status": "ok"}`))
	} else {
		jsonResponse(ctx, 404, ErrorAddressNotFound)
	}
}

func (handler *Handler) tokenIsValid(ctx *fasthttp.RequestCtx) bool {
	return len(handler.Config.Token) == 0 ||
		handler.Config.Token == string(ctx.Request.Header.Peek("Token"))
}

func (handler *Handler) handleUpload(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		jsonResponse(ctx, 405, ErrorMethodNotAllowed)
		return
	}

	if !handler.tokenIsValid(ctx) {
		jsonResponse(ctx, 401, ErrorInvalidToken)
		return
	}

	fileHeader, err := ctx.FormFile("image_file")
	if err != nil {
		jsonResponse(ctx, 400, ErrorImageNotProvided)
		return
	}
	if imageValidated := validateImage(fileHeader); !imageValidated {
		jsonResponse(ctx, 400, ErrorFileIsNotImage)
		return
	}

	imageID := shortid.GetDefault().MustGenerate()
	imagePath := getFilePathFromImageID(handler.Config.DataDir, imageID)
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		panic(err)
	}
	if err := fasthttp.SaveMultipartFile(fileHeader, imagePath); err != nil {
		panic(err)
	}
	jsonResponse(ctx, 200, []byte(fmt.Sprintf(`{"image_id": "%s"}`, imageID)))
}

func (handler *Handler) handleDelete(ctx *fasthttp.RequestCtx) {
	if !ctx.IsDelete() {
		jsonResponse(ctx, 405, ErrorMethodNotAllowed)
		return
	}

	if !handler.tokenIsValid(ctx) {
		jsonResponse(ctx, 401, ErrorInvalidToken)
		return
	}

	match := DeleteRegex.FindSubmatch(ctx.Path())
	if len(match) != 2 {
		jsonResponse(ctx, 404, ErrorAddressNotFound)
		return
	}
	imageID := string(match[1])
	imagePath := getFilePathFromImageID(handler.Config.DataDir, imageID)

	err := os.Remove(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			jsonResponse(ctx, 404, ErrorImageNotFound)
			return
		}
		panic(err)
	}
	jsonResponse(ctx, 204, nil)
}

func (handler *Handler) handleFetch(ctx *fasthttp.RequestCtx) {
	if !ctx.IsGet() {
		jsonResponse(ctx, 405, ErrorMethodNotAllowed)
		return
	}
	options, imageID := parseImageURI(ctx.Path())
	if len(imageID) == 0 {
		jsonResponse(ctx, 404, ErrorAddressNotFound)
		return
	}

	if len(options) == 0 {
		// user wants original file
		imagePath := getFilePathFromImageID(handler.Config.DataDir, imageID)
		if ok := handler.serveFileFromDisk(ctx, imagePath, true); !ok {
			jsonResponse(ctx, 404, ErrorImageNotFound)
		}
		return
	}

	webpAccepted := bytes.Contains(ctx.Request.Header.Peek("accept"), []byte("webp"))

	imageParams, err := createImageParams(
		imageID,
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
		ctx.SetContentType("image/webp")
	} else {
		ctx.SetContentType("image/jpeg")
	}

	cacheFilePath := imageParams.getCachePath(handler.Config.DataDir)
	if ok := handler.serveFileFromDisk(ctx, cacheFilePath, false); ok {
		// request served from cache
		return
	}
	// cache didn't exist

	if err := validateImageParams(imageParams, handler.Config); err != nil {
		errorBody := []byte(fmt.Sprintf(`{"error": "%v"}`, err))
		jsonResponse(ctx, 400, errorBody)
		return
	}

	imagePath := getFilePathFromImageID(handler.Config.DataDir, imageParams.ImageID)

	err = handler.TaskManager.RunTask(imageParams.getMd5(), func() error {
		return convertFunction(imagePath, cacheFilePath, imageParams)
	})

	if err != nil {
		if os.IsNotExist(err) {
			jsonResponse(ctx, 404, ErrorImageNotFound)
			return
		}
		panic(err)
	}

	ctx.Response.SetStatusCode(200)
	handler.serveFileFromDisk(ctx, cacheFilePath, false)
}

func (handler *Handler) handleErrors(ctx *fasthttp.RequestCtx, err error) {
	if _, ok := err.(*fasthttp.ErrSmallBuffer); ok {
		jsonResponse(ctx, 431, []byte(`{"error": "Too big request header"}`))
	} else if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
		jsonResponse(ctx, 408, []byte(`{"error": "Request timeout"}`))
	} else {
		jsonResponse(ctx, 400, []byte(`{"error": "Error when parsing request"}`))
	}
}

func (handler *Handler) serveFileFromDisk(ctx *fasthttp.RequestCtx, filePath string, setContentType bool) bool {
	f, err := os.Open(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println(err)
		}
		return false
	}
	buffer := bytebufferpool.Get()
	defer bytebufferpool.Put(buffer)
	_, err = buffer.ReadFrom(f)
	if err != nil {
		panic(err)
	}
	f.Close()
	ctx.SetBody(buffer.B)
	ctx.Response.Header.SetBytesKV(CacheControlKey, handler.CacheControlHeader)
	if setContentType {
		ctx.SetContentType(http.DetectContentType(buffer.B))
	}
	return true
}

func parseImageURI(requestPath []byte) (options, imageID string) {
	// options are in the format below:
	// w=200,h=200,fit=cover,quality=90

	match := ImageRegex.FindStringSubmatch(string(requestPath))
	if len(match) != 4 {
		return
	}
	options, imageID = match[2], match[3]
	return
}
