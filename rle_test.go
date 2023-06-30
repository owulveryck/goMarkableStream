package main

import (
	"testing"
)

func Test_encodeRLE(t *testing.T) {
	type args struct {
		data []uint8
	}
	tests := []struct {
		name string
		args args
		want []uint8
	}{
		{
			name: "small zeros",
			args: args{
				data: []uint8{0, 0, 0, 0},
			},
			want: []uint8{4, 0},
		},
		{
			name: "small ones",
			args: args{
				data: []uint8{1, 1, 1, 1},
			},
			want: []uint8{4, 1},
		},
		{
			name: "both",
			args: args{
				data: []uint8{0, 0, 0, 0, 1, 1, 1, 1},
			},
			want: []uint8{4, 0, 4, 1},
		},
		{
			name: "both",
			args: args{
				data: []uint8{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3},
			},
			want: []uint8{16, 2, 5, 3},
		},
		{
			name: "lots of zeros",
			args: args{
				data: []uint8{
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0, 0, 0,
				},
			},
			want: []uint8{16, 0, 16, 0, 4, 0},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeRLE(tt.args.data)
			for i, val := range encoded {
				v1, v2 := extractUint4Values(val)
				if tt.want[i*2] != v1 {
					t.Errorf("encodeRLE() count = %v, want %v", v1, tt.want[i*2])
				}
				if tt.want[i*2+1] != v2 {
					t.Errorf("encodeRLE() value = %v, want %v", v2, tt.want[i*2+1])
				}

			}
		})
	}
}

func extractUint4Values(encodedValue uint8) (uint8, uint8) {
	// Mask the upper 4 bits to extract the first value
	value1 := (encodedValue >> 4) & 0x0F

	// Mask the lower 4 bits to extract the second value
	value2 := encodedValue & 0x0F

	return value1 + 1, value2
}
