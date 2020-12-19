package main

import (
	"bytes"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"io/ioutil"
	"mime/multipart"
	"net"
	"os"
	"testing"
)

var TOKEN = []byte("123")

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
	uri, method string,
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
	req := createRequest(uri, method, token, body)
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
	uploadUri := "http://test/upload/"

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
			imagePath:      "./testdata/test.jpg",
			imageParamName: "image_file",
			token:          nil,
			expectedStatus: 405,
			expectedError:  ERROR_METHOD_NOT_ALLOWED,
		},
		{
			name:           "Missing Token",
			method:         "POST",
			imagePath:      "./testdata/test.jpg",
			imageParamName: "image_file",
			token:          nil,
			expectedStatus: 401,
			expectedError:  ERROR_INVALID_TOKEN,
		},
		{
			name:           "Invalid Param Name",
			method:         "POST",
			imagePath:      "./testdata/test.jpg",
			imageParamName: "image_fileee",
			token:          TOKEN,
			expectedStatus: 400,
			expectedError:  ERROR_IMAGE_NOT_PROVIDED,
		},
		{
			name:           "Successful Jpeg Upload",
			method:         "POST",
			imagePath:      "./testdata/test.jpg",
			imageParamName: "image_file",
			token:          TOKEN,
			expectedStatus: 200,
			expectedError:  nil,
		},
		{
			name:           "Successful PNG Upload",
			method:         "POST",
			imagePath:      "./testdata/test.png",
			imageParamName: "image_file",
			token:          TOKEN,
			expectedStatus: 200,
			expectedError:  nil,
		},
		{
			name:           "Successful WEBP Upload",
			method:         "POST",
			imagePath:      "./testdata/test.webp",
			imageParamName: "image_file",
			token:          TOKEN,
			expectedStatus: 200,
			expectedError:  nil,
		},
		{
			name:           "Failed pdf Upload",
			method:         "POST",
			imagePath:      "./testdata/test.pdf",
			imageParamName: "image_file",
			token:          TOKEN,
			expectedStatus: 400,
			expectedError:  ERROR_FILE_IS_NOT_IMAGE,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("Test upload errors %s", tc.name), func(t *testing.T) {
			req := createUploadRequest(
				uploadUri, tc.method, tc.token,
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
