package main

import (
	"image"
	"reflect"
	"testing"
)

func Test_rotate(t *testing.T) {
	const width = 4
	const height = 3

	t.Run("one line", func(t *testing.T) {
		/*
			A B C D
			E F G H
			I J K L

			I E A
			J F B
			K G C
			L H D

			D H L
			C G K
			B F J
			A E I
		*/
		content := []byte{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L'}
		//expected := []byte{'I', 'E', 'A', 'J', 'F', 'B', 'K', 'G', 'C', 'L', 'H', 'D'}
		//expected := []byte{'L', 'H', 'D', 'K', 'G', 'C', 'J', 'F', 'B', 'I', 'E', 'A'}
		expected := []byte{'D', 'H', 'L', 'C', 'G', 'K', 'B', 'F', 'J', 'A', 'E', 'I'}
		expectedIm := image.NewGray(image.Rectangle{Max: image.Point{X: height, Y: width}})
		expectedIm.Pix = expected
		im := image.NewGray(image.Rectangle{Max: image.Point{X: width, Y: height}})
		im.Pix = content
		rotate(im)
		if !reflect.DeepEqual(im, expectedIm) {
			t.Errorf("bad result, expected %s, got %s", expectedIm.Pix, im.Pix)
			t.Errorf("bad result, expected %v, got %v", expectedIm, im)
		}
	})

}
