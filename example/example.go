package main

import (
	"encoding/json"
	"fmt"

	"github.com/pcrawfor/iapverifier"
)

const sharedSecret = "your shared secret here"

func main() {
	verifier := iapverifier.NewVerifier(sharedSecret)

	fmt.Println("Verify receipt bad receipt")
	r, err := verifier.VerifyReceipt("blah", false)
	if err != nil {
		fmt.Println("Error verifying bad receipt:", err)
	}
	fmt.Println("Response for bad receipt:", r)

	fmt.Println("Verify receipt good receipt")

	// you'll need to provide a valid receipt string here
	rec := `your base64 encoded receipt data here`

	r, err = verifier.VerifyReceipt(rec, true)
	fmt.Println("check: ", r)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	if r.IsValid {
		fmt.Println("Receipt is valid, data is: ", r.Data)
		v, _ := json.Marshal(r.Data)
		fmt.Println("JSON: ", string(v))
	} else {
		fmt.Println("Receipt not valid, data is: ", r.Data)
	}
}
