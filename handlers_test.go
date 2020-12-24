package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	bimg "gopkg.in/h2non/bimg.v1"
	"io/ioutil"
	"mime/multipart"
	"net"
	"os"
	"testing"
)

var (
	TOKEN          = []byte("123")
	TEST_FILE_PNG  = "./testdata/test.png"
	TEST_FILE_JPEG = "./testdata/test.jpg"
	TEST_FILE_WEBP = "./testdata/test.webp"
	TEST_FILE_GIF  = "./testdata/test.gif"
	TEST_FILE_PDF  = "./testdata/test.pdf"
)

type UploadResult struct {
	ImageId string `json:"image_id"`
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

func getDefaultConfig() *Config {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		panic(err)
	}
	return &Config{
		DataDir:         dir,
		Token:           string(TOKEN),
		ValidImageSizes: []string{"500x200", "500x500", "100x100"},
	}
}

func TestHealthFunc(t *testing.T) {
	server := CreateServer(&Config{})
	req := fasthttp.AcquireRequest()
	req.SetRequestURI("http://test/health/")
	defer fasthttp.ReleaseRequest(req)

	resp := serve(server, req)

	status := resp.Header.StatusCode()
	if status != 200 {
		t.Errorf("Expected 200 but received %d", status)
	}
	body := string(resp.Body())
	if body != `{"status": "ok"}` {
		t.Errorf("Invalid body: %s", body)
	}
}

func TestUploadFunc(t *testing.T) {
	config := getDefaultConfig()
	server := CreateServer(config)
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
			expectedError:  ERROR_METHOD_NOT_ALLOWED,
		},
		{
			name:           "Missing Token",
			method:         "POST",
			imagePath:      TEST_FILE_JPEG,
			imageParamName: "image_file",
			token:          nil,
			expectedStatus: 401,
			expectedError:  ERROR_INVALID_TOKEN,
		},
		{
			name:           "Invalid Param Name",
			method:         "POST",
			imagePath:      TEST_FILE_JPEG,
			imageParamName: "image_fileee",
			token:          TOKEN,
			expectedStatus: 400,
			expectedError:  ERROR_IMAGE_NOT_PROVIDED,
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
			expectedError:  ERROR_FILE_IS_NOT_IMAGE,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("Test upload errors %s", tc.name), func(t *testing.T) {
			req := createUploadRequest(
				tc.method, tc.token,
				tc.imageParamName, tc.imagePath,
			)
			resp := serve(server, req)
			status := resp.Header.StatusCode()
			body := resp.Body()
			if ct := string(resp.Header.ContentType()); ct != "application/json" {
				t.Fatalf("Expected json response")
			}
			if status != tc.expectedStatus {
				t.Fatalf("Expected %d status but got %d",
					tc.expectedStatus, status)
			}
			if tc.expectedError != nil &&
				!bytes.Equal(body, tc.expectedError) {
				t.Fatalf("Expected %s as error but got %s",
					string(tc.expectedError), string(body))
			}
			if status != 200 {
				errResult := &ErrorResult{}
				err := json.Unmarshal(body, errResult)
				if err != nil || errResult.Error == "" {
					t.Fatalf("Could not parse error: (%s) %v", string(body), err)
				}
			}

		})
	}
}

func TestFetchFunc(t *testing.T) {
	config := getDefaultConfig()
	server := CreateServer(config)
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
			name:           "test png with web accepted false",
			uploadFilePath: TEST_FILE_PNG,
			fetchOpts:      "w=500,h=500,fit=cover",
			webpAccepted:   false,
			expectedStatus: 200,
			expectedError:  nil,
			expectedCt:     CT_JPEG,
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
			expectedCt:     CT_WEBP,
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
			expectedCt:     CT_JPEG,
			expectedWidth:  500,
			expectedHeight: 500,
		},
		{
			name:           "test gif",
			uploadFilePath: TEST_FILE_GIF,
			fetchOpts:      "w=500,h=500,fit=cover",
			webpAccepted:   false,
			expectedStatus: 200,
			expectedError:  nil,
			expectedCt:     CT_GIF,
			expectedWidth:  703,
			expectedHeight: 681,
		},
		{
			name:           "test string as width",
			uploadFilePath: TEST_FILE_JPEG,
			fetchOpts:      "w=hi,h=500,fit=cover",
			webpAccepted:   false,
			expectedStatus: 400,
			expectedError:  []byte(`{"error": "Invalid options: Width should be integer"}`),
			expectedCt:     CT_JSON,
			expectedWidth:  500,
			expectedHeight: 500,
		},
		{
			name:           "test inacceptable dimensions",
			uploadFilePath: TEST_FILE_JPEG,
			fetchOpts:      "w=300,h=200,fit=cover",
			webpAccepted:   false,
			expectedStatus: 403,
			expectedError:  ERROR_INVALID_IMAGE_SIZE,
			expectedCt:     CT_JSON,
			expectedWidth:  0,
			expectedHeight: 0,
		},
	}
	for _, tc := range tt {
		t.Run(fmt.Sprintf("Test upload errors %s", tc.name), func(t *testing.T) {
			uploadReq := createUploadRequest(
				"POST", TOKEN,
				"image_file", tc.uploadFilePath,
			)
			uploadResp := serve(server, uploadReq)

			if uploadResp.Header.StatusCode() != 200 {
				t.Fatalf("Expected upload status 200 but got %d",
					uploadResp.Header.StatusCode())
			}
			uploadResult := &UploadResult{}
			err := json.Unmarshal(uploadResp.Body(), uploadResult)
			if err != nil {
				t.Fatal(err)
			}
			fetchUri := fmt.Sprintf("http://test/image/%s/%s", tc.fetchOpts, uploadResult.ImageId)
			fetchReq := createRequest(fetchUri, "GET", nil, nil)
			if tc.webpAccepted {
				fetchReq.Header.SetBytesKV([]byte("accept"), []byte("webp"))
			}
			fetchResp := serve(server, fetchReq)
			status := fetchResp.Header.StatusCode()
			if status != tc.expectedStatus {
				t.Fatalf("Expected fetch status %d but got %d.",
					tc.expectedStatus, status)
			}
			ct := string(fetchResp.Header.ContentType())
			if ct != tc.expectedCt {
				t.Fatalf("Expected %s as content type but got %s",
					tc.expectedCt, ct)
			}
			body := fetchResp.Body()
			if status != 200 {
				if !bytes.Equal(tc.expectedError, body) {
					t.Fatalf("Expected %s as error but got %s",
						string(tc.expectedError), string(body))
				}
				errResult := &ErrorResult{}
				err := json.Unmarshal(body, errResult)
				if err != nil || errResult.Error == "" {
					t.Fatalf("Could not parse error: (%s) %v", string(body), err)
				}
			} else {
				img := bimg.NewImage(body)
				size, err := img.Size()
				if err != nil {
					t.Fatal(err)
				}
				if size.Width != tc.expectedWidth {
					t.Fatalf("Expected width=%d but is %d", tc.expectedWidth, size.Width)
				}
				if size.Height != tc.expectedHeight {
					t.Fatalf("Expected height=%d but is %d", tc.expectedHeight, size.Height)
				}
			}
		})
	}
}

func Test404(t *testing.T) {
	config := &Config{}
	server := CreateServer(config)
	req := createRequest("http://test/hey", "GET", nil, nil)
	resp := serve(server, req)
	if resp.StatusCode() != 404 {
		t.Fatalf("Expected 404 but got %d", resp.StatusCode())
	}
	ct := string(resp.Header.ContentType())
	if ct != "application/json" {
		t.Fatalf("Expected content type application/json but got %s", ct)
	}
	if !bytes.Equal(resp.Body(), ERROR_ADDRESS_NOT_FOUND) {
		t.Fatalf("Unexpected body: %s", string(resp.Body()))
	}

}

func TestFetchFuncMethodShouldBeGet(t *testing.T) {
	config := getDefaultConfig()
	server := CreateServer(config)
	defer os.RemoveAll(config.DataDir)
	req := createRequest("http://test/image/w=500,h=500/NG4uQBa2f", "POST", nil, nil)
	resp := serve(server, req)
	if resp.StatusCode() != 405 {
		t.Fatalf("Expected 405 but got %d", resp.StatusCode())
	}
}

func TestFetchFuncWithInvalidImageId(t *testing.T) {
	config := getDefaultConfig()
	server := CreateServer(config)
	defer os.RemoveAll(config.DataDir)
	req := createRequest("http://test/image/w=500,h=500/NG4uQBa2f", "GET", nil, nil)
	resp := serve(server, req)
	if resp.StatusCode() != 404 {
		t.Fatalf("Expected 404 but got %d", resp.StatusCode())
	}
	ct := string(resp.Header.ContentType())
	if ct != "application/json" {
		t.Fatalf("Expected content type application/json but got %s", ct)
	}
	if !bytes.Equal(resp.Body(), ERROR_IMAGE_NOT_FOUND) {
		t.Fatalf("Unexpected body: %s", string(resp.Body()))
	}
}

func TestCacheFileIsCreatedAfterFetch(t *testing.T) {
	config := getDefaultConfig()
	server := CreateServer(config)
	defer os.RemoveAll(config.DataDir)
	uploadReq := createUploadRequest(
		"POST", TOKEN,
		"image_file", TEST_FILE_JPEG,
	)
	uploadResp := serve(server, uploadReq)

	if uploadResp.Header.StatusCode() != 200 {
		t.Fatalf("Expected upload status 200 but got %d",
			uploadResp.Header.StatusCode())
	}
	uploadResult := &UploadResult{}
	err := json.Unmarshal(uploadResp.Body(), uploadResult)
	if err != nil {
		t.Fatal(err)
	}
	fetchUri := fmt.Sprintf("http://test/image/w=500,h=500,fit=cover/%s", uploadResult.ImageId)
	fetchReq := createRequest(fetchUri, "GET", nil, nil)
	imageParams := &ImageParams{
		ImageId: uploadResult.ImageId,
		Width:   500,
		Height:  500,
		Quality: config.DefaultImageQuality,
		Fit:     FIT_COVER,
	}
	cachePath := imageParams.GetCachePath(config.DataDir)
	imagePath := ImageIdToFilePath(config.DataDir, uploadResult.ImageId)

	serve(server, fetchReq)
	buf, err := bimg.Read(cachePath)
	if err != nil {
		t.Fatalf("Couldn't read the cache")
	}
	img := bimg.NewImage(buf)
	size, _ := img.Size()
	if size.Width != 500 || size.Height != 500 {
		t.Fatal("Cache is invalid")
	}
	if os.Remove(imagePath) != nil {
		t.Fatal("Couldn't remove image")
	}
	resp := serve(server, fetchReq)
	if resp.StatusCode() != 200 {
		t.Fatal("Server didn't read the cache")
	}

}

func TestDeleteHandler(t *testing.T) {
	config := getDefaultConfig()
	server := CreateServer(config)
	defer os.RemoveAll(config.DataDir)
	uploadReq := createUploadRequest(
		"POST", TOKEN,
		"image_file", TEST_FILE_JPEG,
	)
	uploadResp := serve(server, uploadReq)

	if uploadResp.Header.StatusCode() != 200 {
		t.Fatalf("Expected upload status 200 but got %d",
			uploadResp.Header.StatusCode())
	}
	uploadResult := &UploadResult{}
	err := json.Unmarshal(uploadResp.Body(), uploadResult)
	if err != nil {
		t.Fatal(err)
	}

	tt := []struct {
		name           string
		method         string
		imageId        string
		token          []byte
		expectedStatus int
		expectedBody   []byte
	}{
		{
			name:           "Invalid Method",
			method:         "GET",
			imageId:        "123456789",
			token:          nil,
			expectedStatus: 405,
			expectedBody:   ERROR_METHOD_NOT_ALLOWED,
		},
		{
			name:           "Invalid Address",
			method:         "DELETE",
			imageId:        "123456789",
			token:          nil,
			expectedStatus: 401,
			expectedBody:   ERROR_INVALID_TOKEN,
		},
		{
			name:           "Invalid Address",
			method:         "DELETE",
			imageId:        "123456789/123",
			token:          TOKEN,
			expectedStatus: 404,
			expectedBody:   ERROR_ADDRESS_NOT_FOUND,
		},
		{
			name:           "Invalid Image",
			method:         "DELETE",
			imageId:        "123456789",
			token:          TOKEN,
			expectedStatus: 404,
			expectedBody:   ERROR_IMAGE_NOT_FOUND,
		},
		{
			name:           "Invalid Image",
			method:         "DELETE",
			imageId:        uploadResult.ImageId,
			token:          TOKEN,
			expectedStatus: 204,
			expectedBody:   nil,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("Test upload errors %s", tc.name), func(t *testing.T) {
			uri := fmt.Sprintf("http://test/delete/%s", tc.imageId)
			req := createRequest(uri, tc.method, tc.token, nil)
			resp := serve(server, req)
			status := resp.StatusCode()
			if status != tc.expectedStatus {
				t.Fatalf("Expected %d but got %d", tc.expectedStatus, status)
			}
			ct := string(resp.Header.ContentType())
			if ct != "application/json" {
				t.Fatalf("Expected content type to be application/json but is %s", ct)
			}
			body := resp.Body()
			if !bytes.Equal(body, tc.expectedBody) {
				t.Fatalf("Expected %s as body but got %s",
					string(tc.expectedBody), string(body))
			}
			if body != nil {
				errResult := &ErrorResult{}
				err := json.Unmarshal(body, errResult)
				if err != nil || errResult.Error == "" {
					t.Fatalf("Could not parse error: (%s) %v", string(body), err)
				}
			}
		})
	}

	imagePath := ImageIdToFilePath(config.DataDir, uploadResult.ImageId)
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Fatal("Expected image to be deleted")
	}

}

func TestGettingOriginalImage(t *testing.T) {
	config := getDefaultConfig()
	server := CreateServer(config)

	defer os.RemoveAll(config.DataDir)
	uploadReq := createUploadRequest(
		"POST", TOKEN,
		"image_file", TEST_FILE_PNG,
	)
	uploadResult := &UploadResult{}
	uploadResp := serve(server, uploadReq)
	json.Unmarshal(uploadResp.Body(), uploadResult)
	uri := fmt.Sprintf("http://test/image/%s", "123456789")
	req := createRequest(uri, "GET", nil, nil)
	resp := serve(server, req)
	if resp.StatusCode() != 404 {
		t.Fatalf("Expected 404 but got %d", resp.StatusCode())
	}
	if ct := string(resp.Header.ContentType()); ct != "application/json" {
		t.Fatalf("Expected content type to be application/json but is %s", ct)
	}
	if !bytes.Equal(resp.Body(), ERROR_IMAGE_NOT_FOUND) {
		t.Fatalf("Expected error %s but got %s", string(ERROR_IMAGE_NOT_FOUND), string(resp.Body()))
	}
	uri = fmt.Sprintf("http://test/image/%s", uploadResult.ImageId)
	req = createRequest(uri, "GET", nil, nil)
	resp = serve(server, req)
	if resp.StatusCode() != 200 {
		t.Fatalf("Expected 200 but got %d", resp.StatusCode())
	}
	if ct := string(resp.Header.ContentType()); ct != "image/png" {
		t.Fatalf("Expected content type to be image/png but is %s", ct)
	}
	img := bimg.NewImage(resp.Body())
	size, _ := img.Size()
	if size.Width != 1680 && size.Height != 1050 {
		t.Fatalf("Expected image size to be 1680x1050 but is %dx%d",
			size.Width, size.Height)
	}

}
