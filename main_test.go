package main

import (
	"bytes"
	"io"
	"testing"
)

func Test_getPointer(t *testing.T) {
	type args struct {
		r      io.ReaderAt
		offset int64
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			"ok",
			args{
				r:      bytes.NewReader([]byte{4, 8, 160, 187, 114}),
				offset: 1,
			},
			1924898824,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPointer(tt.args.r, tt.args.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPointer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getPointer() = %v, want %v", got, tt.want)
			}
		})
	}
}
