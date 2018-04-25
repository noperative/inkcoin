/*

A trivial application to illustrate how the blockartlib library can be
used from an application in project 1 for UBC CS 416 2017W2.

Usage:
go run art-app.go
*/

package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import "./blockartlib"

import (
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
<<<<<<< HEAD

	// 	<svg>
	// <path d="M 480 40 L 430 120 L 480 150 L 520 120 H 520 L 480 40" fill="red" stroke="red"></path>
	// <path d="M 420 130 L 350 230 L 480 300 V 160 L 420 130" fill="transparent" stroke="red"> </path>
	// <path d="M 490 160 L 530 140 L 610 240 L 490 300 Z" fill="blue" stroke="blue"></path>
	// <path d="M 761 78 L 741 58 H 711 L 691 78 V 98 L 711 118 H 721 L 758 117 L 770 140 V 160 L 750 180 H 710 L 690 160" fill="transparent" stroke="green"></path>
	// <path d="M 700 40 L 720 200" fill="transparent" stroke="green"></path>
	// <path d="M 720 40 L 740 200" fill="transparent" stroke="green"></path>
	// <path d="M 280 140 L 560 50" fill="transparent" stroke="red"></path>
	// <path d="M 280 140 L 560 50" fill="transpraent" stroke="purple"><path>
	// </svg>
	minerAddr := "127.0.0.1:50417"
=======
	minerAddr := "127.0.0.1:34492"
>>>>>>> ubc/master


	privKeyString := "3081a4020101043069f5ffffd085b51a78166f766330f4771674d8cadfd4bc3556082d59fcaa8d56a74e487e125318c0abb0c71e3852b341a00706052b81040022a164036200043bda4ebb0d9f3d2270e41ce140b889bdb94e889fb7c1b0f082c9919bbb5cde31af295da333e5f216336bface06843b0f5ef7b36d1ab0b28bbe458559b8d48df15763e2e6e955f8102aca1c5e8413a248547ece44bc1be5326debc14cb8add5ed"
	privateKeyBytes, _ := hex.DecodeString(privKeyString)
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytes)
	// TODO: use crypto/ecdsa to read pub/priv keys from a file argument.

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, *privKey)
	if checkError(err) != nil {
		//return
	}

	fmt.Println(canvas)
	fmt.Println(settings)

	validateNum := uint8(6)

	// Add a line.
	shapeHash, blockHash, ink, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 0 5", "transparent", "red")
	if checkError(err) != nil {
		return
	}

	fmt.Println("added a line:", shapeHash, blockHash, ink)
	// Add another line.
	shapeHash2, blockHash2, ink2, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 5 0", "transparent", "blue")
	if checkError(err) != nil {
		return
	}

	fmt.Println("added another line", shapeHash2, blockHash2, ink2)

	fmt.Println("deleting a line!")
	// Delete the first line.
	ink3, err := canvas.DeleteShape(validateNum, shapeHash)
	if checkError(err) != nil {
		return
	}

	fmt.Println("deleted a line", ink3)
	// assert ink3 > ink2

	// Close the canvas.
	ink4, err := canvas.CloseCanvas()
	if checkError(err) != nil {
		return
	}

	fmt.Println("closed canvas", ink4)
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}
