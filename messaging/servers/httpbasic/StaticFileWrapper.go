package httpbasic

import (
	"net/http"
	"os"
	"time"
)

// StaticFSWrapper adds a timestamp to files from embed.
// Without it, static files have the current time, causing cache-control to fail.
// From: https://github.com/golang/go/issues/44854
type StaticFSWrapper struct {
	http.FileSystem
	FixedModTime time.Time
}

func (f *StaticFSWrapper) Open(name string) (http.File, error) {
	file, err := f.FileSystem.Open(name)

	return &StaticFileWrapper{File: file, fixedModTime: f.FixedModTime}, err
}

type StaticFileWrapper struct {
	http.File
	fixedModTime time.Time
}

func (f *StaticFileWrapper) Stat() (os.FileInfo, error) {

	fileInfo, err := f.File.Stat()

	return &StaticFileInfoWrapper{FileInfo: fileInfo, fixedModTime: f.fixedModTime}, err
}

type StaticFileInfoWrapper struct {
	os.FileInfo
	fixedModTime time.Time
}

func (f *StaticFileInfoWrapper) ModTime() time.Time {
	return f.fixedModTime
}
