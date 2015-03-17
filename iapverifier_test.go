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

func TestAllResponses(t *testing.T) {
	r1 := ResponseInfo{0, "Active", true, false, false, nil}
	r2 := ResponseInfo{21000, "App store could not read", false, true, false, nil}
	r3 := ResponseInfo{21002, "Data was malformed", false, true, false, nil}
	r4 := ResponseInfo{21003, "Receipt not authenticated", false, true, false, nil}
	r5 := ResponseInfo{21004, "Shared secret does not match", false, true, false, nil}
	r6 := ResponseInfo{21005, "Receipt server unavailable", false, true, false, nil}
	r7 := ResponseInfo{21006, "Receipt valid but subscription expired", false, false, false, nil}
	r8 := ResponseInfo{21007, "Sandbox receipt sent to Production environment", false, true, true, nil}
	r9 := ResponseInfo{21008, "Production receipt sent to Sandbox environment", false, true, false, nil}
	responses := []ResponseInfo{r1, r2, r3, r4, r5, r6, r7, r8, r9}

	for _, r := range responses {
		appleResp, aerr := runTestWithResponse(r, t)
		if aerr != nil {
			fmt.Println("r:", r)
			t.Error("Apple response error:", aerr)
			break
		}

		if appleResp.StatusCode != r.StatusCode {
			t.Error("StatusCode does not match for:", r.StatusCode)
		}

		if appleResp.Message != r.Message {
			t.Error("Message does not match for:", r.Message)
		}

		if appleResp.Data != r.Data {
			t.Error("Data does not match for:", r.Data)
		}
	}
}

func runTestWithResponse(responseInfo ResponseInfo, t *testing.T) (*ResponseInfo, error) {
	pw := "secret"
	receiptString := "hello"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// send back a response code
		response := map[string]int{"status": responseInfo.StatusCode}
		jsonData, err := json.Marshal(response)

		if err != nil {
			t.Error("Error marshaling response json:", err)
		}

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Accept", "application/json")
		w.Write(jsonData)
	}))
	defer ts.Close()

	v := NewVerifier(pw, RunInSandboxMode())

	// override the host with our test server
	v.Config.Host = ts.URL
	v.Config.Path = ""
	return v.verify(receiptString, false)
}

func TestBasicVerifyReceiptRequest(t *testing.T) {
	pw := "secret"
	receiptString := "hello"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
