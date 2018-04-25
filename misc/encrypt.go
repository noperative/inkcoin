package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"
)

// To encode publicKey use:
// publicKeyBytes, _ = x509.MarshalPKIXPublicKey(&private_key.PublicKey)

// Private Key:
// 3081a40201010430d35b96ee7ced244b5a47de8968b07ecd38a6dd756f0ffb40a72ccd5895e96f24310c1fc544d7f8d026c55213c8fa2ef2a00706052b81040022a164036200040ef0f59ad36a9661ef93044b53e5c2ca2e7b5ce23323367a3428ebeb256716b8c2cfc63225fd88174193cbe13c3137b41719058cd0fabd5713b91bc7b314f8086fba4b29734d675fccd6a7b4a4ec6af96d499ba64d792522f4710791d214ac45

// Public Key:
// 3076301006072a8648ce3d020106052b81040022036200040ef0f59ad36a9661ef93044b53e5c2ca2e7b5ce23323367a3428ebeb256716b8c2cfc63225fd88174193cbe13c3137b41719058cd0fabd5713b91bc7b314f8086fba4b29734d675fccd6a7b4a4ec6af96d499ba64d792522f4710791d214ac45

func main() {
	p384 := elliptic.P384()
	priv1, _ := ecdsa.GenerateKey(p384, rand.Reader)

	privateKeyBytes, _ := x509.MarshalECPrivateKey(priv1)
	encodedPrivateBytes := hex.EncodeToString(privateKeyBytes)

	privateKeyBytesRestored, _ := hex.DecodeString(encodedPrivateBytes)
	priv2, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&priv1.PublicKey)
	encodedPublicBytes := hex.EncodeToString(publicKeyBytes)

	fmt.Println("Public key is: %s", encodedPublicBytes)
	fmt.Println("Private key is: %s", encodedPrivateBytes)
	data := []byte("data")
	// Signing by priv1
	r, s, _ := ecdsa.Sign(rand.Reader, priv1, data)

	// Verifying against priv2 (restored from priv1)
	if !ecdsa.Verify(&priv2.PublicKey, data, r, s) {
		fmt.Printf("Error")
		return
	}

	fmt.Printf("Key was restored from string successfully\n")
	fmt.Printf("go run ink-miner.go 127.0.0.1:12345 %s %s\n", encodedPublicBytes, encodedPrivateBytes)

	// Write key pair to file
	fmt.Println("Start writing to file")
	f, err := os.Create("./key-pairs.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	f.Write([]byte(encodedPrivateBytes))
	f.Write([]byte("\n"))
	f.Write([]byte(encodedPublicBytes))
	f.Write([]byte("\n"))
	f.Close()
	fmt.Println("Finished writing to file")
}
