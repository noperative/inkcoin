package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import "./blockartlib"

import "bufio"
import "fmt"
import "os"
import "crypto/ecdsa"

func main() {
	minerAddr := "127.0.0.1:8080"
	privKeyString := "3081a40201010430d35b96ee7ced244b5a47de8968b07ecd38a6dd756f0ffb40a72ccd5895e96f24310c1fc544d7f8d026c55213c8fa2ef2a00706052b81040022a164036200040ef0f59ad36a9661ef93044b53e5c2ca2e7b5ce23323367a3428ebeb256716b8c2cfc63225fd88174193cbe13c3137b41719058cd0fabd5713b91bc7b314f8086fba4b29734d675fccd6a7b4a4ec6af96d499ba64d792522f4710791d214ac45"
	privateKeyBytes, _ := hex.DecodeString(privKeyString)
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytes)

	// Open a canvas.
	canvas, _, err := blockartlib.OpenCanvas(minerAddr, privKey)
	if checkError(err) != nil {
		return
	}

    validateNum := 2

	// Case L
	shapeHash1, _, _, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 1 0 L 1 2", "transparent", "black")
	if checkError(err) != nil {
		return
	}

	_, err = canvas.DeleteShape(validateNum, shapeHash1)
	if checkError(err) != nil {
		return
	}

	// Case H
	shapeHash2, _, _, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 100 100 H 1", "transparent", "black")
	if err == nil || !err.(blockartlib.ShapeOverlapError) {
		fmt.Println("Expected ShapeOverlapError")
	}

	_, err = canvas.DeleteShape(validateNum, shapeHash2)
	if checkError(err) != nil {
		return
	}
	
	// Case V
	shapeHash3, _, _, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 200 200 V 1", "transparent", "black")
	if checkError(err) != nil {
		return
	}

	_, err = canvas.DeleteShape(validateNum, shapeHash3)
	if checkError(err) != nil {
		return
	}

	// Close the canvas.
	_, err := canvas.CloseCanvas()
	if checkError(err) != nil {
		return
	}
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}
