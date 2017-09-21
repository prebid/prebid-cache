package azure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"hash"
	"net/url"
	"strings"
	"sync"
)

const queryConst = "type=master&ver=1.0&sig="
const queryConstSize = len(queryConst)

func newAuthorization(key string) (*authorization, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}

	return &authorization{
		signaturePool: sync.Pool{
			New: func() interface{} {
				var sigBytes [128]byte
				copy(sigBytes[0:queryConstSize], queryConst)
				return &signatureData{
					hashInstance: hmac.New(sha256.New, decodedKey),
					sigBytes:     sigBytes,
				}
			},
		},
	}, nil
}

func (auth *authorization) sign(verb, resourceType, resourceLink, date string) (string, error) {
	cachedData := auth.signaturePool.Get().(*signatureData)
	defer auth.free(cachedData)

	if signature, err := cachedData.sign(verb, resourceType, resourceLink, date); err != nil {
		return "", err
	} else {
		return signature, nil
	}
}

func (auth *authorization) free(data *signatureData) {
	data.reset()
	auth.signaturePool.Put(data)
}

// authorization struct just consolidates the object pools
type authorization struct {
	signaturePool sync.Pool // Stores *signatureData instances
}

type signatureData struct {
	hashInstance hash.Hash
	shaSum       [64]byte
	sigBytes     [128]byte
	strToSign    [128]byte
}

func (data *signatureData) sign(verb, resourceType, resourceLink, date string) (string, error) {
	strToSign := data.strToSign[0:0]
	strToSign = append(strToSign, strings.ToLower(verb)...)
	strToSign = append(strToSign, '\n')
	strToSign = append(strToSign, resourceType...)
	strToSign = append(strToSign, '\n')
	strToSign = append(strToSign, resourceLink...)
	strToSign = append(strToSign, '\n')
	strToSign = append(strToSign, strings.ToLower(date)...)
	strToSign = append(strToSign, '\n', '\n')

	if _, err := data.hashInstance.Write(strToSign); err != nil {
		return "", errors.New("Failed to write strToSign into the hash. " + err.Error())
	}

	shaSum := data.hashInstance.Sum(data.shaSum[0:0])

	encodedLen := base64.StdEncoding.EncodedLen(len(shaSum))
	sigBytes := data.sigBytes[0:queryConstSize]
	for len(sigBytes) < (queryConstSize + encodedLen) {
		sigBytes = append(sigBytes, '0')
	}

	base64.StdEncoding.Encode(sigBytes[queryConstSize:], shaSum)
	return url.QueryEscape(string(sigBytes[0 : queryConstSize+encodedLen])), nil
}

func (data *signatureData) reset() {
	data.hashInstance.Reset()
}
