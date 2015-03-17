package iapverifier

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSandboxModeSetCorreclty(t *testing.T) {
	v := NewVerifier("secret", RunInSandboxMode())
	if !v.isSandboxMode() {
		t.Error("Should be set to run in sandbox mode")
	}
}

func TestBasicVerifyReceipt(t *testing.T) {
	pw := "secret"
	receiptString := "hello"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("request:", r)
		reqData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error("Error reading request body:", err)
		}

		d := map[string]string{}
		perr := json.Unmarshal(reqData, &d)
		if perr != nil {
			t.Error("Error parsing request data:", perr)
		}

		if d["password"] != pw {
			t.Error("Expected password to be", pw, "got:", d["password"])
		}

		// decode receipt data
		receiptData := d["receipt-data"]
		decoded, derr := base64.StdEncoding.DecodeString(receiptData)
		if derr != nil {
			t.Error("Error decoding receipt-data:", derr)
		}

		if string(decoded) != receiptString {
			t.Error("Expected receipt string to be", receiptData, "got:", decoded)
		}
	}))
	defer ts.Close()

	v := NewVerifier(pw, RunInSandboxMode())

	if !v.isSandboxMode() {
		t.Error("Should be set to run in sandbox mode")
	}

	// override the host with our test server
	v.Config.Host = ts.URL
	v.Config.Path = ""
	v.verify(receiptString, false)
}
