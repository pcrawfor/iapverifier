# iapverifier

iapverifier is a Go library for processing Apple's In App Purchase Receipt data.

Based on a similar library I implemented in javascript as an npm module.  This library processes receipt data by communicating with Apple's receipt verification services.

It interprets the status codes and provides a full solution for taking raw receipt data from an iap and determining if it is a valid purchase. 
