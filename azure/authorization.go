package azure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

func newAuthorization(key string) *authorization {
	return &authorization{
		key: key,
	}
}

func (auth *authorization) sign(verb, resourceType, resourceLink, date string) string {
	strToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n\n",
		strings.ToLower(verb),
		resourceType,
		resourceLink,
		strings.ToLower(date),
	)

	decodedKey, _ := base64.StdEncoding.DecodeString(auth.key)
	sha256 := hmac.New(sha256.New, []byte(decodedKey))
	sha256.Write([]byte(strToSign))

	signature := base64.StdEncoding.EncodeToString(sha256.Sum(nil))
	u := url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", signature))

	return u
}

// authorization struct just consolidates the object pools
type authorization struct {
	key string
}
