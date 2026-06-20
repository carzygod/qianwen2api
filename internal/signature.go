package internal

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func GenerateSignature(eoCltDvidn, ve, kp, requestParamValue, sacsft string, reqt int64) string {
	signText := eoCltDvidn + ve + kp + requestParamValue
	signSalt := fmt.Sprintf("%s:%d", sacsft, reqt)

	h := hmac.New(sha256.New, []byte(signSalt))
	h.Write([]byte(signText))
	signature := h.Sum(nil)

	return base64.StdEncoding.EncodeToString(signature)
}

func GenerateBodySignature(body []byte, secret string) string {
	if len(body) == 0 || secret == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func GenerateQwenWebSignature(eoCltDvidn, ve, kp, requestParamValue, bodySign, sacsft string, reqt int64) string {
	signText := eoCltDvidn + ve + kp + requestParamValue + bodySign
	signSalt := fmt.Sprintf("%s:%d", sacsft, reqt)

	h := hmac.New(sha256.New, []byte(signSalt))
	h.Write([]byte(signText))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
