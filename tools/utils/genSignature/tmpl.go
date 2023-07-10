package main

const tmpl = `
package main

import (
	"crypto/md5"
	"fmt"
)

func compareSig(src []byte, sig [16]byte) bool {
	if len(src) != 16 {
		return false
	}
	for i := 0; i < 16; i++ {
		if src[i] != sig[i] {
			return false
		}
	}
	return true
}

func is{{ .Name }}(content []byte) bool {
	sig := []byte{ {{ .Sig }} }
	return compareSig(sig, md5.Sum(content[{{ .Start }}:{{ .End }}]))
}
`
