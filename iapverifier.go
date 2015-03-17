/*

iapverifier

Author: 	Paul Crawford
License: 	(refer to license file)

*/

package iapverifier

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

/*
  responseCodes:
    0:     { message:"Active", valid: true, error: false }
    21000: { message:"App store could not read", valid: false, error: true }
    21002: { message:"Data was malformed", valid: false, error: true }
    21003: { message:"Receipt not authenticated", valid: false, error: true }
    21004: { message:"Shared secret does not match", valid: false, error: true }
    21005: { message:"Receipt server unavailable", valid: false, error: true }
    21006: { message:"Receipt valid but sub expired", valid: false, error: false }
    21007: { message:"Sandbox receipt sent to Production environment", valid: false, error: true, redirect: true } # special case for app review handling - forward any request that is intended for the Sandbox but was sent to Production, this is what the app review team does
    21008: { message:"Production receipt sent to Sandbox environment", valid: false, error: true }
*/

const productionHost = "https://buy.itunes.apple.com"
const sandboxHost = "https://sandbox.itunes.apple.com"
const retryCode = 21007

// ResponseInfo represents a message indicating the meaning of the IAP response code and flags indicating whether it is a valid response or error
// Flag indicating whether the request should be redirected to another IAP environment.
type ResponseInfo struct {
	StatusCode int
	Message    string
	IsValid    bool
	IsError    bool
	IsRedirect bool
	Data       interface{}
}

// AppleResponse represents the response data from Apple
type AppleResponse struct {
	Status  int         `json:"status"`
	Receipt interface{} `json:"receipt,omitempty"`
}

// IAP Responses codes
var responseCodes = map[int]ResponseInfo{
	0:     ResponseInfo{0, "Active", true, false, false, nil},
	21000: ResponseInfo{21000, "App store could not read", false, true, false, nil},
	21002: ResponseInfo{21002, "Data was malformed", false, true, false, nil},
	21003: ResponseInfo{21003, "Receipt not authenticated", false, true, false, nil},
	21004: ResponseInfo{21004, "Shared secret does not match", false, true, false, nil},
	21005: ResponseInfo{21005, "Receipt server unavailable", false, true, false, nil},
	21006: ResponseInfo{21006, "Receipt valid but subscription expired", false, false, false, nil},
	21007: ResponseInfo{21007, "Sandbox receipt sent to Production environment", false, true, true, nil},
	21008: ResponseInfo{21008, "Production receipt sent to Sandbox environment", false, true, false, nil},
}

// HostConfig contains the configuration for making an outgoing request to a host
type HostConfig struct {
	Host   string
	Port   int
	Path   string
	Method string
}

// IapVerifier is the core verifier object
type IapVerifier struct {
	Config HostConfig
	Secret string
}

// NewVerifier initializes a Verifier with the given itunes shared secret, defaults to production server
func NewVerifier(secret string, options ...func(*IapVerifier) error) *IapVerifier {
	v := IapVerifier{}
	v.Config = HostConfig{Host: productionHost, Port: 443, Path: "/verifyReceipt", Method: "POST"}
	v.Secret = secret

	for _, option := range options {
		err := option(&v)
		if err != nil {
			return nil
		}
	}

	return &v
}

// RunInSandboxMode - sets the client to run against the Apple Sandbox server, can be passed as an init arg
func RunInSandboxMode() func(*IapVerifier) error {
	return func(v *IapVerifier) error {
		v.runInSandboxMode()
		return nil
	}
}

func (v *IapVerifier) runInSandboxMode() {
	v.Config.Host = sandboxHost
}

// RunInProductionMode - sets the client to run against the Apple Production server
func (v *IapVerifier) runInProductionMode() {
	v.Config.Host = productionHost
}

// IsSandboxMode - checks whether the client is running in sandbox mode
func (v *IapVerifier) isSandboxMode() bool {
	return v.Config.Host == sandboxHost
}

// VerifyReceipt - verifies the receipt against Apple's servers
// The receipt parameter is expected to be the raw receipt received by your payment observer in iOS, the verifier will convert it to base64 encoding.
func (v *IapVerifier) VerifyReceipt(receipt string, isBase64Encoded bool) (*ResponseInfo, error) {
	return v.verifyWithRetry(receipt, isBase64Encoded)
}

// verifyWithRetry
// Verify the receipt data with retry logic, if the Apple response is 21007 indicating that Sandbox data was sent to
// the Apple Production servers, retry in the sandbox environment
func (v *IapVerifier) verifyWithRetry(receipt string, isBase64Encoded bool) (*ResponseInfo, error) {
	r, err := v.verify(receipt, isBase64Encoded)

	// Retry if the Apple response code indicates that we should and we are not in Sandbox mode already
	if r.StatusCode == retryCode && !v.isSandboxMode() {
		fmt.Println("Retry in sandbox environment")
		v.runInSandboxMode()
		if v.isSandboxMode() {
			r, err = v.verify(receipt, isBase64Encoded)
		}
		v.runInProductionMode() // switch back to production mode
		return r, err
	}

	return r, err
}

// verify - Verify the receipt data from via the Apple servers set in Config.Host
func (v *IapVerifier) verify(receipt string, isBase64Encoded bool) (*ResponseInfo, error) {
	// TODO: encode the receipt string in base64
	var encodedReceipt string
	if isBase64Encoded {
		encodedReceipt = receipt
	} else {
		var buf bytes.Buffer
		encoder := base64.NewEncoder(base64.StdEncoding, &buf)
		encoder.Write([]byte(receipt))
		encoder.Close()
		encodedReceipt = buf.String()
	}

	data := map[string]string{
		"receipt-data": encodedReceipt,
		"password":     v.Secret,
	}

	// JSON encode the data map
	jsonData, err := json.Marshal(data)

	// Set the request headers
	url := v.Config.Host + v.Config.Path
	req, err := http.NewRequest(v.Config.Method, url, bytes.NewBuffer(jsonData))
	req.ContentLength = int64(len(jsonData))
	req.Header.Add("Content-Type", "application/json")

	// Make an https request to Apple
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error response from apple: ", err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		fmt.Println("Http Error Code: ", res.StatusCode)
		return nil, errors.New("Error getting apple data")
	}

	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		// TODO: handle error case
		return nil, err
	}
	res.Body.Close()

	appleResp := AppleResponse{}

	err = json.Unmarshal(resData, &appleResp)
	if err != nil {
		return nil, err
	}

	r := responseCodes[appleResp.Status]
	r.Data = appleResp.Receipt
	return &r, nil
}
