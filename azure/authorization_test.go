package azure

import (
	"testing"
	"time"
)

func TestValidSignature(t *testing.T) {
	auth := newAuthorization("aGprc2NoNzc2MjdlZHVpSHVER1NIQ0pld3lhNzMyNjRlN2ReIyQmI25jc2Fr")
	sometime := time.Unix(123, 345).UTC()
	formatedTime := sometime.Format("Mon, 02 Jan 2006 15:04:05 GMT")

	signature := auth.sign("POST", "docs", "/dbs/prebidcache/colls/cache/docs", formatedTime)
	expected := "type%3Dmaster%26ver%3D1.0%26sig%3Db3cssh4LYbBDNUIWqAIfIxgbwllUao1BpLwUI8TT%2FCo%3D"

	if signature != expected {
		t.Errorf("Bad signature. Expected: %s, Got: %s", expected, signature)
	}
}

func BenchmarkSignature(b *testing.B) {
	auth := newAuthorization("aGprc2NoNzc2MjdlZHVpSHVER1NIQ0pld3lhNzMyNjRlN2ReIyQmI25jc2Fr")
	sometime := time.Unix(123, 345).UTC()
	formatedTime := sometime.Format("Mon, 02 Jan 2006 15:04:05 GMT")

	for i := 0; i < b.N; i++ {
		auth.sign("POST", "docs", "/dbs/prebidcache/colls/cache/docs", formatedTime)
	}
}