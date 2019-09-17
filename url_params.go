package main

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
)

var re *regexp.Regexp

func init() {
	re = regexp.MustCompile("/image/(?P<Name>[0-9a-f]{32})/(?P<Width>[0-9]00)/(?P<Height>[0-9]00)/(?P<Type>cover|contain)(?P<WEBP>.\\w+)?$")
}

var (
	COVER = []byte("cover")
	WEBP  = []byte(".webp")
)

type ImageParams struct {
	FileName string
	Width    int
	Height   int
	Crop     bool
	Webp     bool
}

func GetParamsFromUri(reqUri []byte) (*ImageParams, error) {
	match := re.FindSubmatch(reqUri)
	if len(match) != 6 {
		return nil, errors.New("Not Match")
	}
	width, _ := strconv.Atoi(string(match[2]))
	height, _ := strconv.Atoi(string(match[3]))
	params := &ImageParams{
		FileName: string(match[1]),
		Width:    width,
		Height:   height,
		Crop:     bytes.Equal(match[4], COVER),
		Webp:     bytes.Equal(match[5], WEBP),
	}
	return params, nil
}
