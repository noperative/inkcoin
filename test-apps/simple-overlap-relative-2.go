package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import "./blockartlib"

import "bufio"
import "fmt"
import "os"
import "crypto/ecdsa"

func main() {
	minerAddr := "127.0.0.1:8081"
	privKeyString := "3081a40201010430abb996d825e0a92b470d34f506eca5294a9198922ca4b18941d83150ceb4fe919b2bc6e5fb0c98deb727bcbd431e2de9a00706052b81040022a1640362000468e7eb38ea0d892ee056dd19d5c0358b91ae1995130886215f733375b0f52033b9584aa77fc4cbf8dace46b7f8f603cafeca0927956fcb6c72bb829cf04f7287faf09e9c96304925547178039bc4c7b17d94d51de01a98060e755bffdc8015bd"
	privateKeyBytes, _ := hex.DecodeString(privKeyString)
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytes)

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, privKey)
	if checkError(err) != nil {
		return
	}

	success := true
    validateNum := 2

	// Case l
	// Add an overlapping line (perpendicular)
	_, _, _, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 100 101 l 2 0", "transparent", "black")
	checkShapeOverlap(err)

	// Add an overlapping line (parallel)
	_, _, _, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 101 100 l 0 1", "transparent", "black")
	checkShapeOverlap(err)

	// Case h
	// Add an overlapping line (perpendicular)
	_, _, _, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 101 100 v 2", "transparent", "black")
	checkShapeOverlap(err)

	// Add an overlapping line (parallel)
	_, _, _, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 100 101 h 1", "transparent", "black")
	
	checkShapeOverlap(err)
	
	// Case v
	// Add an overlapping line (perpendicular)
	_, _, _, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 200 201 h 2", "transparent", "black")
	
	checkShapeOverlap(err)

	// Add an overlapping line (parallel)
	_, _, _, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 201 200 v 1", "transparent", "black")
	checkShapeOverlap(err)

	// Close the canvas.
	ink4, err := canvas.CloseCanvas()
	if checkError(err) != nil {
		return
	}

	if success {
		fmt.Println("Test case passed")
	} else {
		fmt.Println("Test case failed")
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

func checkShapeOverlap(err error) { 
	if err == nil || !err.(blockartlib.ShapeOverlapError) {
		fmt.Println("Expected ShapeOverlapError")
		success = false
	}
}
