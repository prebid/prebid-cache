package azure

import (
	"testing"
	"time"
)

func TestValidSignature(t *testing.T) {
	auth := newAuthorization("abc")
	sometime := time.Unix(123, 345).UTC()
	formatedTime := sometime.Format("Mon, 02 Jan 2006 15:04:05 GMT")

	signature := auth.sign("POST", "docs", "/dbs/prebidcache/colls/cache/docs", formatedTime)
	expected := "type%3Dmaster%26ver%3D1.0%26sig%3D%2FV6RDQKxak0tC5KFaPtcEAQOOp%2BMFppHYg7%2BRcVN5ec%3D"

	if signature != expected {
		t.Errorf("Bad signature. Expected: %s, Got: %s", expected, signature)
	}
}
