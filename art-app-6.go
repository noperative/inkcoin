/*

This art-app is purely used to generate the HTML file

*/

package main

// Expects blockartlib.go to be in the ../blockartlib/ dir, relative to
// this art-app.go file
import (
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"./blockartlib"
)

////// TYPES FOR THE WEBSERVER ///////
type AddRequest struct {
	Fill   string `json:"fill"`
	Stroke string `json:"stroke"`
	Path   string `json:"path"`
}

type AddResponse struct {
	SVGString    string `json:"svg-string"`
	InkRemaining uint32 `json:"ink-remaining"`
	ShapeHash    string `json:"shape-hash"`
	BlockHash    string `json:"block-hash"`
}

type HistoryResponse struct {
	Paths []string `json:"paths"`
}

////// END OF TYPES FOR THE WEBSERVER ///////

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
	fmt.Println("OpenCanvas")
	checkError(err)

	fmt.Println(canvas)
	fmt.Println(settings)

	generateHTML(canvas, settings)

	// Close the canvas.
	ink1, err := canvas.CloseCanvas()
	fmt.Println("CloseCanvas")
	checkError(err)
	fmt.Println("%d", ink1)

	// Reopen canvas to poll for blockchain
	canvas, _, err = blockartlib.OpenCanvas(minerAddr, *privKey)

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			tpl, err := template.ParseFiles("blockart.html", "paths.html")

			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, err.Error(), 500)
			}

			err = tpl.ExecuteTemplate(w, "blockart.html", nil)
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, err.Error(), 500)
			}
		} else if r.Method == "POST" {
			var addReq AddRequest

			err = json.NewDecoder(r.Body).Decode(&addReq)
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println("Error Marshalling/Decoding")
				http.Error(w, err.Error(), 500)
				return
			}

			shapeHash, blockHash, ink, err := canvas.AddShape(4, blockartlib.PATH, addReq.Path, addReq.Fill, addReq.Stroke)
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, err.Error(), 500)
				return
			}

			svgString, err := canvas.GetSvgString(shapeHash)
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, err.Error(), 500)
				return
			}

			addResp := AddResponse{
				SVGString:    svgString,
				InkRemaining: ink,
				ShapeHash:    shapeHash,
				BlockHash:    blockHash}

			resp, err := json.Marshal(addResp)
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			// w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}

	// Serve the html file if its a GET
	http.HandleFunc("/", handler)

	log.Fatal(http.ListenAndServe(":8888", nil))

	// go func(canvas blockartlib.Canvas) {
	// 	for {
	// 		reader := bufio.NewReader(os.Stdin)
	// 		fmt.Println("For AddShape: ADD,[PATH],[FILL],[STROKE],[PATH|CIRCLE]")
	// 		fmt.Println("For DeleteShape: DELETE,[SHAPEHASH]")
	// 		fmt.Print("Enter text: ")
	// 		text, _ := reader.ReadString('\n')
	// 		fmt.Println(text)
	// 		tokens := strings.Split(text, ",")
	// 		OPTYPE := tokens[0]

	// 		fmt.Printf("Tokens: %+v\n", tokens)
	// 		validateNum := uint8(4)
	// 		if OPTYPE == "ADD" {
	// 			path := tokens[1]
	// 			fill := tokens[2]
	// 			stroke := tokens[3]

	// 			var pathType blockartlib.ShapeType
	// 			if tokens[4] == "PATH" {
	// 				pathType = blockartlib.PATH
	// 			} else if tokens[4] == "CIRCLE" {
	// 				pathType = blockartlib.CIRCLE
	// 			} else {
	// 				continue
	// 			}

	// 			fmt.Println("Adding from command line: ")
	// 			shapeHash, blockHash, inkRemaining, err := canvas.AddShape(validateNum, pathType, path, fill, stroke)

	// 			fmt.Println("Adding completed: ")
	// 			if err != nil {
	// 				checkError(err)
	// 			} else {
	// 				fmt.Println("Add is: %s, BlockHash: %s, InkRemaining: %d", shapeHash, blockHash, inkRemaining)
	// 			}
	// 		} else if OPTYPE == "DELETE" {
	// 			shapeHash := tokens[1]
	// 			fmt.Println("Deleting from command line: ")
	// 			_, err := canvas.DeleteShape(validateNum, shapeHash)
	// 			fmt.Println("Deleting completed: ")
	// 			if err != nil {
	// 				checkError(err)
	// 			}
	// 		}
	// 	}
	// }(canvas)

	for {
	}
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
func generateHTML(canvas blockartlib.Canvas, settings blockartlib.CanvasSettings) {
	// Create a blank HTML file
	HTML, err := os.Create("./art-app.html")
	checkError(err)
	dir, err := os.Getwd()
	fmt.Println("Currently working directory is: %s", dir)
	pathsHTML, err := os.Create("./paths.html")
	checkError(err)
	defer HTML.Close()
	defer pathsHTML.Close()

	// Append starting HTML tags
	pre := []byte("<!DOCTYPE html>\n<html>\n<head>\n\t<title>HTML SVG Output</title>\n</head>\n")
	bodyString := fmt.Sprintf("<body>\n\t<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" version=\"1.1\">\n", settings.CanvasXMax, settings.CanvasYMax)
	body := []byte(bodyString)
	HTML.Write(pre)
	HTML.Write(body)

	// Get the longest blockchain
	// Start with the genesis block and recursively add to chain
	gHash, err := canvas.GetGenesisBlock()
	fmt.Println("GetGenesisBlock")
	checkError(err)
	blockchain := getLongestBlockchain(gHash, canvas)
	// svgPaths := make([]string, 0)
	// Add the HTML SVG string of each opeartion in the blockchain
	fmt.Println("GetShapes")
	for _, bHash := range blockchain {
		sHashes, err := canvas.GetShapes(bHash)
		checkError(err)
		for _, sHash := range sHashes {
			HTMLSVGString, err := canvas.GetSvgString(sHash)
			// Expect to see an InvalidShapeHashError
			// as the first line was deleted, but art-node can
			// never tell strictly by shapeHash
			if err == nil {
				fmt.Println("Writing to paths.HTML")
				HTML.Write([]byte("\t\t" + HTMLSVGString + "\n"))
				pathsHTML.Write([]byte(HTMLSVGString + "\n"))
			} else {
				fmt.Println("Error in svg string")
				break
			}
		}
	}

	// Append ending HTML tags
	suf := []byte("\t</svg>\n</body>\n</html>\n")
	HTML.Write(suf)
}
