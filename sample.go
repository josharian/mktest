package main

import "errors"

type File struct {
}

func (f *File) Read(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) Write(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}
