# iapverifier

iapverifier is a Go library for processing Apple's In App Purchase Receipt data.

Based on a similar library I implemented in javascript as an npm module.  This library processes receipt data by communicating with Apple's receipt verification services.

It interprets the status codes and provides a full solution for taking raw receipt data from an iap and determining if it is a valid purchase.

# Usage:

    package main

    import (
      "encoding/json"
      "fmt"
      "github.com/pcrawfor/iapverifier"
    )

    func main() {
      receipt := "some apple receipt string"
      verifier := iapverifier.NewVerifier("secret")      
      response, err := verifier.VerifyReceipt(receipt, false)
                 
      if err != nil {
        fmt.Println("Error: ", err)
      }
      if response.IsValid {
        fmt.Println("Receipt is valid, data is: ", response.Data)
        v, _ := json.Marshal(response.Data)
        fmt.Println("JSON: ", string(v))
      } else {
        fmt.Println("Receipt not valid, data is: ", response.Data)
      }
    } 
