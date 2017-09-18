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
				return &signatureData{
					hashInstance: hmac.New(sha256.New, decodedKey),
					sigBytes:     append([]byte(nil), queryConst...),
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
	shaSum       []byte
	sigBytes     []byte
	strToSign    []byte
}

func (data *signatureData) sign(verb, resourceType, resourceLink, date string) (string, error) {
	data.strToSign = append(data.strToSign, strings.ToLower(verb)...)
	data.strToSign = append(data.strToSign, '\n')
	data.strToSign = append(data.strToSign, resourceType...)
	data.strToSign = append(data.strToSign, '\n')
	data.strToSign = append(data.strToSign, resourceLink...)
	data.strToSign = append(data.strToSign, '\n')
	data.strToSign = append(data.strToSign, strings.ToLower(date)...)
	data.strToSign = append(data.strToSign, '\n', '\n')

	if _, err := data.hashInstance.Write(data.strToSign); err != nil {
		return "", errors.New("Failed to write strToSign into the hash. " + err.Error())
	}

	data.shaSum = data.hashInstance.Sum(data.shaSum)

	encodedLen := base64.StdEncoding.EncodedLen(len(data.shaSum))
	for len(data.sigBytes) < (queryConstSize + encodedLen) {
		data.sigBytes = append(data.sigBytes, '0')
	}

	base64.StdEncoding.Encode(data.sigBytes[queryConstSize:], data.shaSum)
	return url.QueryEscape(string(data.sigBytes[0 : queryConstSize+encodedLen])), nil
}

func (data *signatureData) reset() {
	data.hashInstance.Reset()
	data.sigBytes = data.sigBytes[0:queryConstSize]
	data.shaSum = data.shaSum[0:0]
	data.strToSign = data.strToSign[0:0]
}
