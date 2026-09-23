package hex

import (
	"fmt"
	"net/http"
)

func ParseTargetHash(r *http.Request) ([32]byte, error) {
	hashHex := r.URL.Query().Get("hash")
	if hashHex != "" {
		return decodeHex32(hashHex)
	}

	return [32]byte{}, fmt.Errorf("missing 'hash' query parameter")
}

func fromHexChar(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	default:
		return 255
	}
}

func decodeHex32(src string) ([32]byte, error) {
	var dst [32]byte
	if len(src) != 64 {
		return dst, fmt.Errorf("invalid hash hex length: expected 64 hex characters")
	}
	for i := range 32 {
		hi := fromHexChar(src[2*i])
		lo := fromHexChar(src[2*i+1])
		if hi == 255 || lo == 255 {
			return dst, fmt.Errorf("invalid hex character")
		}
		dst[i] = (hi << 4) | lo
	}
	return dst, nil
}
