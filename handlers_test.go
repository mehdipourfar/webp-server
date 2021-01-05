package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/matryer/is"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	bimg "gopkg.in/h2non/bimg.v1"
	"io/ioutil"
	"mime/multipart"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

var (
	TOKEN          = []byte("123")
	TEST_FILE_PNG  = "./testdata/test.png"
	TEST_FILE_JPEG = "./testdata/test.jpg"
	TEST_FILE_WEBP = "./testdata/test.webp"
	TEST_FILE_PDF  = "./testdata/test.pdf"
)

type UploadResult struct {
	ImageID string `json:"image_id"`
}

type ErrorResult struct {
	Error string `json:"error"`
}

func createRequest(uri, method string, token []byte, body *bytes.Buffer) *fasthttp.Request {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(uri)
	if method != "GET" {
		req.Header.SetMethod(method)
	}
	if token != nil {
		req.Header.SetBytesKV([]byte("Token"), token)
	}
	if body != nil {
		req.SetBody(body.Bytes())
	}
	return req
}

func createUploadRequest(
	method string,
	token []byte,
	paramName, path string,
) *fasthttp.Request {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	fi, err := file.Stat()
	if err != nil {
		panic(err)
	}
	file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, fi.Name())
	if err != nil {
		panic(err)
	}
	_, err = part.Write(fileContents)
	if err != nil {
		panic(err)
	}
	ct := writer.FormDataContentType()
	err = writer.Close()
	if err != nil {
		panic(err)
	}
	req := createRequest("http://test/upload/", method, token, body)
	req.Header.SetContentType(ct)
	return req
}

func serve(server *fasthttp.Server, req *fasthttp.Request) *fasthttp.Response {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		err := server.Serve(ln)
		if err != nil {
			panic(fmt.Errorf("failed to serve: %v", err))
		}
	}()

	client := fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	resp := fasthttp.AcquireResponse()
	err := client.Do(req, resp)
	if err != nil {
		panic(err)
	}
	return resp
}

func getTestConfig() *Config {
	cfg := getDefaultConfig()
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		panic(err)
	}
	cfg.DataDir = dir
	cfg.Token = string(TOKEN)
	cfg.DefaultImageQuality = 90
	cfg.ValidImageSizes = []string{"500x200", "500x500", "100x100"}
	cfg.ValidImageQualities = []int{80, 90, 95, 100}
	return cfg
}

func TestHealthFunc(t *testing.T) {
	is := is.New(t)
	server := createServer(&Config{})
	req := fasthttp.AcquireRequest()
	req.SetRequestURI("http://test/health/")
	defer fasthttp.ReleaseRequest(req)

	resp := serve(server, req)
	is.Equal(resp.Header.StatusCode(), 200)
	is.Equal(resp.Body(), []byte(`{"status": "ok"}`))
}

func TestUploadFunc(t *testing.T) {
	is := is.New(t)
	config := getTestConfig()
	server := createServer(config)
	defer os.RemoveAll(config.DataDir)

	tt := []struct {
		name           string
		method         string
		imagePath      string
		imageParamName string
		token          []byte
		expectedStatus int
		expectedError  []byte
	}{
		{
			name:           "Incorrect Method",
			method:         "GET",
			imagePath:      TEST_FILE_JPEG,
			imageParamName: "image_file",
			token:          nil,
			expectedStatus: 405,
			expectedError:  ErrorMethodNotAllowed,
		},
		{
			name:           "Missing Token",
			method:         "POST",
			imagePath:      TEST_FILE_JPEG,
			imageParamName: "image_file",
			token:          nil,
			expectedStatus: 401,
			expectedError:  ErrorInvalidToken,
		},
		{
			name:           "Invalid Param Name",
			method:         "POST",
			imagePath:      TEST_FILE_JPEG,
			imageParamName: "image_fileee",
			token:          TOKEN,
			expectedStatus: 400,
			expectedError:  ErrorImageNotProvided,
		},
		{
			name:           "Successful Jpeg Upload",
			method:         "POST",
			imagePath:      TEST_FILE_JPEG,
			imageParamName: "image_file",
			token:          TOKEN,
			expectedStatus: 200,
			expectedError:  nil,
		},
		{
			name:           "Successful PNG Upload",
			method:         "POST",
			imagePath:      TEST_FILE_PNG,
			imageParamName: "image_file",
			token:          TOKEN,
			expectedStatus: 200,
			expectedError:  nil,
		},
		{
			name:           "Successful WEBP Upload",
			method:         "POST",
			imagePath:      TEST_FILE_WEBP,
			imageParamName: "image_file",
			token:          TOKEN,
			expectedStatus: 200,
			expectedError:  nil,
		},
		{
			name:           "Failed pdf Upload",
			method:         "POST",
			imagePath:      TEST_FILE_PDF,
			imageParamName: "image_file",
			token:          TOKEN,
			expectedStatus: 400,
			expectedError:  ErrorFileIsNotImage,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("Test upload errors %s", tc.name), func(t *testing.T) {
			is := is.NewRelaxed(t)
			req := createUploadRequest(
				tc.method, tc.token,
				tc.imageParamName, tc.imagePath,
			)
			resp := serve(server, req)
			body := resp.Body()
			is.Equal(resp.Header.ContentType(), []byte("application/json"))
			is.Equal(resp.Header.StatusCode(), tc.expectedStatus)
			if tc.expectedError != nil {
				is.Equal(body, tc.expectedError)
			}
			if resp.Header.StatusCode() != 200 {
				errResult := &ErrorResult{}
				err := json.Unmarshal(body, errResult)
				is.NoErr(err)
				is.True(errResult.Error != "")
			}
		})
	}
}

func TestFetchFunc(t *testing.T) {
	config := getTestConfig()
	server := createServer(config)
	defer os.RemoveAll(config.DataDir)
	tt := []struct {
		name           string
		uploadFilePath string
		fetchOpts      string
		webpAccepted   bool
		expectedStatus int
		expectedError  []byte
		expectedCt     string
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "test png with webp accepted false",
			uploadFilePath: TEST_FILE_PNG,
			fetchOpts:      "w=500,h=500,fit=cover",
			webpAccepted:   false,
			expectedStatus: 200,
			expectedError:  nil,
			expectedCt:     "image/jpeg",
			expectedWidth:  500,
			expectedHeight: 500,
		},
		{
			name:           "test png with webp accepted true",
			uploadFilePath: TEST_FILE_PNG,
			fetchOpts:      "w=500,h=500,fit=cover",
			webpAccepted:   true,
			expectedStatus: 200,
			expectedError:  nil,
			expectedCt:     "image/webp",
			expectedWidth:  500,
			expectedHeight: 500,
		},
		{
			name:           "test webp with webp accepted false",
			uploadFilePath: TEST_FILE_WEBP,
			fetchOpts:      "w=500,h=500,fit=cover",
			webpAccepted:   false,
			expectedStatus: 200,
			expectedError:  nil,
			expectedCt:     "image/jpeg",
			expectedWidth:  500,
			expectedHeight: 500,
		},
		{
			name:           "test string as width",
			uploadFilePath: TEST_FILE_JPEG,
			fetchOpts:      "w=hi,h=500,fit=cover",
			webpAccepted:   false,
			expectedStatus: 400,
			expectedError:  []byte(`{"error": "Invalid options: Width should be integer"}`),
			expectedCt:     "application/json",
			expectedWidth:  500,
			expectedHeight: 500,
		},
		{
			name:           "test inacceptable dimensions",
			uploadFilePath: TEST_FILE_JPEG,
			fetchOpts:      "w=300,h=200,fit=cover",
			webpAccepted:   false,
			expectedStatus: 400,
			expectedError:  []byte(`{"error": "size=300x200 is not supported by server. Contact server admin."}`),
			expectedCt:     "application/json",
			expectedWidth:  0,
			expectedHeight: 0,
		},
		{
			name:           "test inacceptable quality",
			uploadFilePath: TEST_FILE_JPEG,
			fetchOpts:      "w=500,h=500,q=60",
			webpAccepted:   false,
			expectedStatus: 400,
			expectedError:  []byte(`{"error": "quality=60 is not supported by server. Contact server admin."}`),
			expectedCt:     "application/json",
			expectedWidth:  0,
			expectedHeight: 0,
		},
		{
			name:           "acceptable quality",
			uploadFilePath: TEST_FILE_JPEG,
			fetchOpts:      "w=500,h=500,q=80",
			webpAccepted:   false,
			expectedStatus: 200,
			expectedError:  nil,
			expectedCt:     "image/jpeg",
			expectedWidth:  500,
			expectedHeight: 313,
		},
	}
	for _, tc := range tt {
		t.Run(fmt.Sprintf("Test upload errors %s", tc.name), func(t *testing.T) {
			is := is.NewRelaxed(t)
			uploadReq := createUploadRequest(
				"POST", TOKEN,
				"image_file", tc.uploadFilePath,
			)
			uploadResp := serve(server, uploadReq)
			is.Equal(uploadResp.Header.StatusCode(), 200)
			uploadResult := &UploadResult{}
			err := json.Unmarshal(uploadResp.Body(), uploadResult)
			is.Equal(err, nil)
			fetchUri := fmt.Sprintf("http://test/image/%s/%s", tc.fetchOpts, uploadResult.ImageID)
			fetchReq := createRequest(fetchUri, "GET", nil, nil)
			if tc.webpAccepted {
				fetchReq.Header.SetBytesKV([]byte("accept"), []byte("webp"))
			}
			fetchResp := serve(server, fetchReq)
			status := fetchResp.Header.StatusCode()
			is.Equal(status, tc.expectedStatus)
			is.Equal(string(fetchResp.Header.ContentType()), tc.expectedCt)
			body := fetchResp.Body()
			if status != 200 {
				is.Equal(tc.expectedError, body)
				errResult := &ErrorResult{}
				err := json.Unmarshal(body, errResult)
				is.NoErr(err)
				is.True(errResult.Error != "")
			} else {
				img := bimg.NewImage(body)
				size, err := img.Size()
				is.NoErr(err)
				is.Equal(size.Width, tc.expectedWidth)
				is.Equal(size.Height, tc.expectedHeight)
			}
		})
	}
}

func Test404(t *testing.T) {
	is := is.New(t)
	config := &Config{}
	server := createServer(config)
	req := createRequest("http://test/hey", "GET", nil, nil)
	resp := serve(server, req)
	is.Equal(resp.StatusCode(), 404)
	is.Equal(string(resp.Header.ContentType()), "application/json")
	is.Equal(resp.Body(), ErrorAddressNotFound)
	req = createRequest("http://test/image/w=500/", "GET", nil, nil)
	resp = serve(server, req)
	is.Equal(resp.StatusCode(), 404)
}

func TestFetchFuncMethodShouldBeGet(t *testing.T) {
	is := is.New(t)
	config := getTestConfig()
	server := createServer(config)
	defer os.RemoveAll(config.DataDir)
	req := createRequest("http://test/image/w=500,h=500/NG4uQBa2f", "POST", nil, nil)
	resp := serve(server, req)
	is.Equal(resp.StatusCode(), 405)
}

func TestFetchFuncWithInvalidImageID(t *testing.T) {
	is := is.New(t)
	config := getTestConfig()
	server := createServer(config)
	defer os.RemoveAll(config.DataDir)
	req := createRequest("http://test/image/w=500,h=500/NG4uQBa2f", "GET", nil, nil)
	resp := serve(server, req)
	is.Equal(resp.StatusCode(), 404)
	is.Equal(string(resp.Header.ContentType()), "application/json")
	is.Equal(resp.Body(), ErrorImageNotFound)
}

func TestCacheFileIsCreatedAfterFetch(t *testing.T) {
	is := is.New(t)
	config := getTestConfig()
	server := createServer(config)
	defer os.RemoveAll(config.DataDir)
	uploadReq := createUploadRequest(
		"POST", TOKEN,
		"image_file", TEST_FILE_JPEG,
	)
	uploadResp := serve(server, uploadReq)

	is.Equal(uploadResp.Header.StatusCode(), 200)
	uploadResult := &UploadResult{}
	err := json.Unmarshal(uploadResp.Body(), uploadResult)
	is.NoErr(err)
	fetchUri := fmt.Sprintf("http://test/image/w=500,h=500,fit=cover/%s", uploadResult.ImageID)
	fetchReq := createRequest(fetchUri, "GET", nil, nil)
	imageParams := &ImageParams{
		ImageID: uploadResult.ImageID,
		Width:   500,
		Height:  500,
		Quality: config.DefaultImageQuality,
		Fit:     FitCover,
	}
	cachePath := imageParams.getCachePath(config.DataDir)
	imagePath := getFilePathFromImageID(config.DataDir, uploadResult.ImageID)

	serve(server, fetchReq)
	buf, err := bimg.Read(cachePath)
	is.NoErr(err)
	img := bimg.NewImage(buf)
	size, _ := img.Size()
	is.Equal(size.Width, 500)
	is.Equal(size.Height, 500)
	is.NoErr(os.Remove(imagePath))
	resp := serve(server, fetchReq)
	is.Equal(resp.StatusCode(), 200)
}

func TestDeleteHandler(t *testing.T) {
	is := is.New(t)
	config := getTestConfig()
	server := createServer(config)
	defer os.RemoveAll(config.DataDir)
	uploadReq := createUploadRequest(
		"POST", TOKEN,
		"image_file", TEST_FILE_JPEG,
	)
	uploadResp := serve(server, uploadReq)

	is.Equal(uploadResp.Header.StatusCode(), 200)
	uploadResult := &UploadResult{}
	err := json.Unmarshal(uploadResp.Body(), uploadResult)
	is.NoErr(err)
	tt := []struct {
		name           string
		method         string
		imageID        string
		token          []byte
		expectedStatus int
		expectedBody   []byte
	}{
		{
			name:           "Invalid Method",
			method:         "GET",
			imageID:        "123456789",
			token:          nil,
			expectedStatus: 405,
			expectedBody:   ErrorMethodNotAllowed,
		},
		{
			name:           "Invalid Address",
			method:         "DELETE",
			imageID:        "123456789",
			token:          nil,
			expectedStatus: 401,
			expectedBody:   ErrorInvalidToken,
		},
		{
			name:           "Invalid Address",
			method:         "DELETE",
			imageID:        "123456789/123",
			token:          TOKEN,
			expectedStatus: 404,
			expectedBody:   ErrorAddressNotFound,
		},
		{
			name:           "Invalid Image",
			method:         "DELETE",
			imageID:        "123456789",
			token:          TOKEN,
			expectedStatus: 404,
			expectedBody:   ErrorImageNotFound,
		},
		{
			name:           "Valid Image",
			method:         "DELETE",
			imageID:        uploadResult.ImageID,
			token:          TOKEN,
			expectedStatus: 204,
			expectedBody:   nil,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("Test delete errors %s", tc.name), func(t *testing.T) {
			is := is.NewRelaxed(t)
			uri := fmt.Sprintf("http://test/delete/%s", tc.imageID)
			req := createRequest(uri, tc.method, tc.token, nil)
			resp := serve(server, req)
			is.Equal(resp.StatusCode(), tc.expectedStatus)
			is.Equal(string(resp.Header.ContentType()), "application/json")
			body := resp.Body()
			is.Equal(body, tc.expectedBody)
			if body != nil {
				errResult := &ErrorResult{}
				err := json.Unmarshal(body, errResult)
				is.Equal(err, nil)
				is.True(errResult.Error != "")
			}
		})
	}

	imagePath := getFilePathFromImageID(config.DataDir, uploadResult.ImageID)
	_, err = os.Stat(imagePath)
	is.True(os.IsNotExist(err))

}

func TestGettingOriginalImage(t *testing.T) {
	is := is.New(t)
	config := getTestConfig()
	server := createServer(config)

	defer os.RemoveAll(config.DataDir)
	uploadReq := createUploadRequest(
		"POST", TOKEN,
		"image_file", TEST_FILE_PNG,
	)
	uploadResult := &UploadResult{}
	uploadResp := serve(server, uploadReq)
	err := json.Unmarshal(uploadResp.Body(), uploadResult)
	is.NoErr(err)
	uri := fmt.Sprintf("http://test/image/%s", "123456789")
	req := createRequest(uri, "GET", nil, nil)
	resp := serve(server, req)
	is.Equal(resp.StatusCode(), 404)
	is.Equal(string(resp.Header.ContentType()), "application/json")
	is.Equal(resp.Body(), ErrorImageNotFound)
	uri = fmt.Sprintf("http://test/image/%s", uploadResult.ImageID)
	req = createRequest(uri, "GET", nil, nil)
	resp = serve(server, req)
	is.Equal(resp.StatusCode(), 200)
	is.Equal(string(resp.Header.ContentType()), "image/png")
	img := bimg.NewImage(resp.Body())
	size, _ := img.Size()
	is.Equal(size.Width, 1680)
	is.Equal(size.Height, 1050)

}

func TestConcurentConversionRequests(t *testing.T) {
	is := is.New(t)
	config := getTestConfig()
	server := createServer(config)

	defer os.RemoveAll(config.DataDir)
	uploadReq := createUploadRequest(
		"POST", TOKEN,
		"image_file", TEST_FILE_PNG,
	)
	uploadResult := &UploadResult{}
	uploadResp := serve(server, uploadReq)
	err := json.Unmarshal(uploadResp.Body(), uploadResult)
	is.NoErr(err)

	var wg sync.WaitGroup
	reqUri := fmt.Sprintf("http://test/image/w=500,h=500,fit=cover/%s", uploadResult.ImageID)

	var functionCalls int64

	// override convertFunction which is used in handleFetch api
	convertFunction = func(inputPath, outputPath string, params *ImageParams) error {
		atomic.AddInt64(&functionCalls, 1)
		return convert(inputPath, outputPath, params)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchReq := createRequest(reqUri, "GET", nil, nil)
			resp := serve(server, fetchReq)
			is.Equal(resp.StatusCode(), 200)
		}()
	}
	wg.Wait()
	is.Equal(functionCalls, int64(1))
}

func TestAllSizesAndQualitiesAreAvailableWhenDebugging(t *testing.T) {
	is := is.New(t)
	config := getTestConfig()
	server := createServer(config)

	defer os.RemoveAll(config.DataDir)
	uploadReq := createUploadRequest(
		"POST", TOKEN,
		"image_file", TEST_FILE_PNG,
	)
	uploadResult := &UploadResult{}
	uploadResp := serve(server, uploadReq)
	err := json.Unmarshal(uploadResp.Body(), uploadResult)
	is.NoErr(err)
	uri := fmt.Sprintf("http://test/image/w=800,h=900,q=72/%s", uploadResult.ImageID)
	req := createRequest(uri, "GET", nil, nil)
	resp := serve(server, req)
	is.Equal(resp.StatusCode(), 400)
	config.Debug = true
	resp = serve(server, req)
	is.Equal(resp.StatusCode(), 200)
}
