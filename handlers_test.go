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

func serve(handler fasthttp.RequestHandler, req *fasthttp.Request) *fasthttp.Response {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		err := fasthttp.Serve(ln, handler)
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

func getHandler() *Handler {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		panic(err)
	}
	config := &Config{
		DATA_DIR:    dir,
		TOKEN:       string(TOKEN),
		VALID_SIZES: []string{"500x200", "500x500", "100x100"},
	}
	handler := &Handler{
		Config: config,
	}
	return handler
}

func TestHealthFunc(t *testing.T) {
	handler := &Handler{}
	req := fasthttp.AcquireRequest()
	req.SetRequestURI("http://test/health/")
	defer fasthttp.ReleaseRequest(req)

	resp := serve(handler.handleRequests, req)

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
	handler := getHandler()
	defer os.RemoveAll(handler.Config.DATA_DIR)

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
			resp := serve(handler.handleRequests, req)
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
		})
	}
}

func TestFetchFunc(t *testing.T) {
	handler := getHandler()
	defer os.RemoveAll(handler.Config.DATA_DIR)
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
			name:           "test png with original format",
			uploadFilePath: TEST_FILE_PNG,
			fetchOpts:      "w=500,h=500,fit=cover,format=original",
			webpAccepted:   true,
			expectedStatus: 200,
			expectedError:  nil,
			expectedCt:     CT_PNG,
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
			name:           "test webp with webp accepted false and original format",
			uploadFilePath: TEST_FILE_WEBP,
			fetchOpts:      "w=500,h=500,fit=cover,format=original",
			webpAccepted:   false,
			expectedStatus: 200,
			expectedError:  nil,
			expectedCt:     CT_WEBP,
			expectedWidth:  500,
			expectedHeight: 500,
		},
		{
			name:           "test string as width",
			uploadFilePath: TEST_FILE_JPEG,
			fetchOpts:      "w=hi,h=500,fit=cover,format=original",
			webpAccepted:   false,
			expectedStatus: 400,
			expectedError:  []byte(`{"error": "Invalid options: Width should be integer"}`),
			expectedCt:     CT_JSON,
			expectedWidth:  500,
			expectedHeight: 500,
		},
	}
	for _, tc := range tt {
		t.Run(fmt.Sprintf("Test upload errors %s", tc.name), func(t *testing.T) {
			uploadReq := createUploadRequest(
				"POST", TOKEN,
				"image_file", tc.uploadFilePath,
			)
			uploadResp := serve(handler.handleRequests, uploadReq)

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
			fetchResp := serve(handler.handleRequests, fetchReq)
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
