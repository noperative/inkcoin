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
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	// Read file content and cast to string
	ipPortBytes, err := ioutil.ReadFile("./ip-ports.txt")
	checkError(err)
	ipPortString := string(ipPortBytes[:])

	keyPairsBytes, err := ioutil.ReadFile("./key-pairs.txt")
	checkError(err)
	keyPairsString := string(keyPairsBytes[:])

	// Parse ip-port and privKey from content string
	minerAddr := strings.Split(ipPortString, "\n")[0]
	privKeyString := strings.Split(keyPairsString, "\n")[0]
	privKeyBytes, err := hex.DecodeString(privKeyString)
	checkError(err)
	privKey, err := x509.ParseECPrivateKey(privKeyBytes)
	checkError(err)

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, *privKey)
	checkError(err)

	fmt.Println(canvas)
	fmt.Println(settings)

	validateNum := uint8(3)

	// Jan's SVG -- Part 1
	svg5 := "M 850 850 L 850 50 L 50 50 L 50 850 L 850 850 M 750 750 L 150 750 L 150 150 L 750 150 L 750 750"
	_, _, _, err = canvas.AddShape(validateNum, blockartlib.PATH, svg5, "darkgrey", "darkgrey")
	checkError(err)
	svg6 := "M 885 885 L 885 15 L 15 15 L 15 885 L 885 885 M 860 860 L 40 860 L 40 40 L 860 40 L 860 860"
	_, _, _, err = canvas.AddShape(validateNum, blockartlib.PATH, svg6, "#232323", "darkgrey")
	checkError(err)
	svg7 := "M 550 674 L 700 449 L 550 224 H 350 L 200 449 L 350 674 H 550 M 540 654 L 678 449 L 540 244 H 360 L 222 449 L 360 654 H 540"
	_, _, _, err = canvas.AddShape(validateNum, blockartlib.PATH, svg7, "#666666", "darkgrey")
	checkError(err)

	// Close the canvas.
	ink1, err := canvas.CloseCanvas()
	checkError(err)

	fmt.Println("%d", ink1)
}

// If error is non-nil, print it out.
func checkError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ", err.Error())
	}
}

// Recursively get the longest blockchain
func getLongestBlockchain(currBlockHash string, canvas blockartlib.Canvas) []string {
	// Add current block hash to longest chain
	longestBlockchain := []string{}
	longestBlockchain = append(longestBlockchain, currBlockHash)

	// Iterate through children of current block if any exist,
	// Adding the longest of them all to the longest blockchain
	children, err := canvas.GetChildren(currBlockHash)
	checkError(err)

	longestChildBlockchain := []string{}
	for _, child := range children {
		childBlockchain := getLongestBlockchain(child, canvas)
		if len(childBlockchain) > len(longestChildBlockchain) {
			longestChildBlockchain = childBlockchain
		}
	}

	return append(longestBlockchain, longestChildBlockchain...)
}

// Generate an HTML file, filled exclusively with 
// HTML SVG strings from the longest blockchain in canvas
func generateHTML(canvas blockartlib.Canvas) {
	// Create a blank HTML file
	HTML, err := os.Create("./art-app-1.html")
	checkError(err)
	defer HTML.Close()

	// Append starting HTML tags
	pre := []byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<!DOCTYPE html>\n<html>\n<head>\n\t<title>HTML SVG Output</title>\n</head>\n")
	body := []byte("<body>\n\t<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"900\" height=\"900\" version=\"1.1\">\n")
	HTML.Write(pre)
	HTML.Write(body)

	// Get the longest blockchain
	// Start with the genesis block and recursively add to chain
	gHash, err := canvas.GetGenesisBlock()
	checkError(err)
	blockchain := getLongestBlockchain(gHash, canvas)

	// Add the HTML SVG string of each opeartion in the blockchain
	for _, bHash := range blockchain {
		sHashes, err := canvas.GetShapes(bHash)
		checkError(err)
		for _, sHash := range sHashes {
			HTMLSVGString, err := canvas.GetSvgString(sHash)
			// Expect to see an InvalidShapeHashError
			// as the first line was deleted, but art-node can
			// never tell strictly by shapeHash
			if err == nil {
				HTML.Write([]byte("\t\t" + HTMLSVGString + "\n"))
			} else {
				break
			}
		}
	}

	// Append ending HTML tags
	suf := []byte("\t</svg>\n</body>\n</html>\n")
	HTML.Write(suf)
}
