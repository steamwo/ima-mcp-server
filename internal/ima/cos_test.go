package ima

import "testing"

func TestBuildCOSAuthorization(t *testing.T) {
	t.Parallel()
	headers := map[string]string{
		"content-length": "123", "host": "example-1250000000.cos.ap-guangzhou.myqcloud.com",
	}
	got := buildCOSAuthorization("AKIDEXAMPLE", "SECRETEXAMPLE", "PUT", "/abc/report.pdf", headers, 1700000000, 1700003600)
	const want = "q-sign-algorithm=sha1&q-ak=AKIDEXAMPLE&q-sign-time=1700000000;1700003600&q-key-time=1700000000;1700003600&q-header-list=content-length;host&q-url-param-list=&q-signature=112df927e6ac9a98013a3db3a1df82ef708678f0"
	if got != want {
		t.Fatalf("authorization mismatch\n got: %s\nwant: %s", got, want)
	}
}
