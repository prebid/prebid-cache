package azure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"
)

func TestValidSignature(t *testing.T) {
	auth, err := newAuthorization("aGprc2NoNzc2MjdlZHVpSHVER1NIQ0pld3lhNzMyNjRlN2ReIyQmI25jc2Fr")
	if err != nil {
		t.Errorf("Couldn't get an auth instance. %v", err)
		return
	}
	sometime := time.Unix(123, 345).UTC()
	formatedTime := sometime.Format("Mon, 02 Jan 2006 15:04:05 GMT")

	signature, err := auth.sign("POST", "docs", "/dbs/prebidcache/colls/cache/docs", formatedTime)
	if err != nil {
		t.Errorf("Failed to generate a signature. %v", err)
		return
	}
	// This was chosen experimentally from a working version of the code, before refactors and optimizations
	expected := "type%3Dmaster%26ver%3D1.0%26sig%3Db3cssh4LYbBDNUIWqAIfIxgbwllUao1BpLwUI8TT%2FCo%3D"

	if signature != expected {
		t.Errorf("Bad signature. Expected: %s, Got: %s", expected, signature)
	}
}

// TestTwoSignatures makes sure that we can generate two signatures successfully.
func TestTwoSignatures(t *testing.T) {
	TestValidSignature(t)
	TestValidSignature(t)
}

// TestNewPoolValue makes sure that new data from the
func TestNewPoolValue(t *testing.T) {
	key := "aGprc2NoNzc2MjdlZHVpSHVER1NIQ0pld3lhNzMyNjRlN2ReIyQmI25jc2Fr"
	auth, _ := newAuthorization(key)
	newData := auth.signaturePool.Get().(*signatureData)

	assertEmpty(t, newData)
}

// TestClearedState makes sure that the mutable data taken from the sync pool is "empty" after
// a call to sign()
func TestClearedState(t *testing.T) {
	key := "aGprc2NoNzc2MjdlZHVpSHVER1NIQ0pld3lhNzMyNjRlN2ReIyQmI25jc2Fr"
	decodedKey, _ := base64.StdEncoding.DecodeString(key)
	auth, _ := newAuthorization(key)
	seededData := &signatureData{
		hashInstance: hmac.New(sha256.New, decodedKey),
		sigBytes:     append([]byte(nil), queryConst...),
	}
	auth.signaturePool.Put(seededData)

	sometime := time.Unix(123, 345).UTC()
	formatedTime := sometime.Format("Mon, 02 Jan 2006 15:04:05 GMT")
	auth.sign("POST", "docs", "/dbs/prebidcache/colls/cache/docs", formatedTime)
	assertEmpty(t, seededData)
	if cap(seededData.sigBytes) < 70 {
		t.Errorf("sigBytes should have room for at least 70 elements. Got %d", cap(seededData.sigBytes))
	}
	if cap(seededData.sigBytes) < 30 {
		t.Errorf("shaSum should have room for at least 30 elements. Got %d", cap(seededData.shaSum))
	}
	if cap(seededData.sigBytes) == 90 {
		t.Errorf("strToSign should have room for at least 90 elements. Got %d", cap(seededData.strToSign))
	}
}

func assertEmpty(t *testing.T, data *signatureData) {
	if string(data.sigBytes) != queryConst {
		t.Errorf("SeededData.sigBytes should match the queryConst value. Got %v", data.sigBytes)
	}

	if len(data.strToSign) != 0 {
		t.Errorf("SeededData.strToSign should be empty. Got %v", data.strToSign)
	}

	if len(data.shaSum) != 0 {
		t.Errorf("SeededData.shaSum should be empty. Got %v", data.shaSum)
	}
}

func BenchmarkSignature(b *testing.B) {
	auth, _ := newAuthorization("aGprc2NoNzc2MjdlZHVpSHVER1NIQ0pld3lhNzMyNjRlN2ReIyQmI25jc2Fr")
	sometime := time.Unix(123, 345).UTC()
	formatedTime := sometime.Format("Mon, 02 Jan 2006 15:04:05 GMT")

	for i := 0; i < b.N; i++ {
		auth.sign("POST", "docs", "/dbs/prebidcache/colls/cache/docs", formatedTime)
	}
}
