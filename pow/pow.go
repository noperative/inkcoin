package pow

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"math/rand"

	"../blockchain"
)

// Return true if hex representation of hash has exactly N trailing zeroes
func Verify(hash string, N int) bool {
	l := len(hash)
	return strings.Count(hash[l-N:], "0") == N && strings.Count(hash[l-N-1:], "0") == N
}

func Solve(block blockchain.Block, powDiff uint8, start uint32, solved chan blockchain.Block, done chan bool) {
	h := md5.New()
	N := int(powDiff)
	//fmt.Println("starting operation with start point: ", start)
	for {
		select {
		case <-done:
			//fmt.Println("job done, stopping")
			return
		default:
			block.Nonce = rand.Uint32()
			bytes, _ := json.Marshal(block)
			h.Write(bytes)
			hash := hex.EncodeToString(h.Sum(nil))
			if Verify(hash, N) {
				defer Recover()
				solved <- block
				return
			} else {
				h.Reset()
			}
		}
	}
}

func Recover() {
	// recover from panic caused by writing to a closed channel
	if r := recover(); r != nil {
		fmt.Println("recovered from closed channel")
		return
	}
}
