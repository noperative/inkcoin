package miner

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"
	"../blockchain"
	"../libminer"
	"../minerserver"
	"../pow"
	"../utils"
)

const (
	TRANSPARENT = "transparent"
)

// Our singleton miner instance
var MinerInstance *Miner

// Primitive representation of active art miners
var ArtNodeList map[int]bool = make(map[int]bool)

// List of peers WE connect TO, not peers that connect to US
var PeerList map[string]*Peer = make(map[string]*Peer)

var BlockCond *sync.Cond

const (
	// Global TTL of propagate requests
	TTL = 2
	// Maximum threads we will use for problem solving
	MAX_THREADS = 1
	// Num new blocks with no operation before repropagating op
	BLOCKS_BEFORE_REPROPAGATE = 10
)

// Global blockchain Parent->Children Map
// Key-value pairs are added when a child arrives but its parent has yet to arrive
// Key: Parent not yet in the blockchain
// Val: List of orphaned children, representing their indices in BlockNodeArray
var ParentHashMap map[string][]int = make(map[string][]int)

// Global block chain array
var BlockNodeArray []blockchain.BlockNode

// Global block chain search map
// Key: The hash of a block
// Val: The index of block with such hash in BlockNodeArray
var BlockHashMap map[string]int = make(map[string]int)

// Map to keep track of longest paths
// Key: The hash of the block
// Val: Each element contains the path up until itself (inclusive) and the len of the path
var PathMap map[string]LongestPathInfo = make(map[string]LongestPathInfo)

// Locks for local blockchain and blockchainmap
// BlockChainMutex only allows concurrent R or single W
// BlockArrayMutex only protects W
// ParentMapMutex only allows concurrent R or single W
var (
	BlockChainMutex *sync.RWMutex
	BlockArrayMutex *sync.Mutex
	ParentMapMutex  *sync.RWMutex
	PathMapMutex    *sync.RWMutex
)

// Current Job ID
var CurrJobId int = 0

// This lock is to guarantee operations are unique, even if they have the same svgString, fill and stroke
var (
	OpNum   uint64 = 0
	OpMutex *sync.Mutex
)

/*******************************
| Structs for the miners to use internally
| note: shared structs should be put in a different lib
********************************/
type Miner struct {
	PrivKey    *ecdsa.PrivateKey
	Addr       net.Addr
	Settings   minerserver.MinerNetSettings
	InkAmt     int
	LMI        *LibMinerInterface
	MSI        *MinerServerInterface
	BlockChain []blockchain.BlockNode
	POpChan    chan PropagateOpArgs
	PBlockChan chan PropagateBlockArgs
	SOpChan    chan blockchain.Operation
	SBlockChan chan blockchain.Block
}

type MinerInfo struct {
	Address net.Addr
	Key     ecdsa.PublicKey
}

type LibMinerInterface struct {
	SOpChan chan blockchain.OperationInfo
	POpChan chan PropagateOpArgs
}

type MinerServerInterface struct {
	Client *rpc.Client
}

type Peer struct {
	Client        *rpc.Client
	LastHeartBeat time.Time
}

// For calculating the longest path
type LongestPathInfo struct {
	Len  int                // Length of the block
	Path []blockchain.Block // The longest path of blocks excluding the current block
}

/*******************************
| Miner functions
********************************/
func (m *Miner) ConnectToServer(ip string) {
	miner_server_int := new(MinerServerInterface)

	LocalAddr, err := net.ResolveTCPAddr("tcp", ":0")
	CheckError(err, "ConnectToServer:ResolveLocalAddr")

	ServerAddr, err := net.ResolveTCPAddr("tcp", ip)
	CheckError(err, "ConnectToServer:ResolveServerAddr")

	conn, err := net.DialTCP("tcp", LocalAddr, ServerAddr)
	CheckError(err, "ConnectToServer:DialTCP")

	fmt.Println("ConnectToServer::connecting to server on:", conn.LocalAddr().String())

	client := rpc.NewClient(conn)
	miner_server_int.Client = client
	m.MSI = miner_server_int
}

/*******************************
| Lib->Miner RPC functions
********************************/

// Setup an interface that implements rpc calls for the lib
func OpenLibMinerConn(ip string, pop chan PropagateOpArgs, sop chan blockchain.OperationInfo) {
	lib_miner_int := &LibMinerInterface{sop, pop}
	server := rpc.NewServer()
	server.Register(lib_miner_int)

	tcp, err := net.Listen("tcp", ip)
	CheckError(err, "OpenLibMinerConn:Listen")

	fmt.Println("Start writing ip:port to file")
	f, err := os.Create("./ip-ports.txt")
	_ = CheckError(err, "OpenLibMinerConn:os.Create")
	f.Write([]byte(tcp.Addr().String()))
	f.Write([]byte("\n"))
	f.Close()
	fmt.Println("Finished writing to file")

	MinerInstance.LMI = lib_miner_int

	fmt.Println("OpenLibMinerConn:: Listening on: ", tcp.Addr().String())
	server.Accept(tcp)
}

func (lmi *LibMinerInterface) OpenCanvas(req *libminer.Request, response *libminer.RegisterResponse) (err error) {
	if Verify(req.Msg, req.HashedMsg, req.R, req.S, MinerInstance.PrivKey) {
		//Generate an id in a basic fashion
		for i := 0; ; i++ {
			if !ArtNodeList[i] {
				ArtNodeList[i] = true
				response.Id = i
				response.CanvasXMax = MinerInstance.Settings.CanvasSettings.CanvasXMax
				response.CanvasYMax = MinerInstance.Settings.CanvasSettings.CanvasYMax
				break
			}
		}
		return nil
	}

	err = fmt.Errorf("invalid user")
	return err
}

func (lmi *LibMinerInterface) GetInk(req *libminer.Request, response *libminer.InkResponse) (err error) {
	if Verify(req.Msg, req.HashedMsg, req.R, req.S, MinerInstance.PrivKey) {
		MinerInstance.InkAmt = CalculateInk(utils.GetPublicKeyString(MinerInstance.PrivKey.PublicKey))
		response.InkRemaining = uint32(MinerInstance.InkAmt)
		return nil
	}

	err = fmt.Errorf("invalid user")
	return err
}


func (lmi *LibMinerInterface) Draw(req *libminer.Request, response *libminer.DrawResponse) (err error) {
	if Verify(req.Msg, req.HashedMsg, req.R, req.S, MinerInstance.PrivKey) {
		MinerInstance.InkAmt = CalculateInk(utils.GetPublicKeyString(MinerInstance.PrivKey.PublicKey))
		var drawReq libminer.DrawRequest
		json.Unmarshal(req.Msg, &drawReq)
		pubKeyString := utils.GetPublicKeyString(MinerInstance.PrivKey.PublicKey)

		// Create Operation
		OpMutex.Lock()
		op := blockchain.Operation{
			OpType:    blockchain.ADD,
			SVGString: drawReq.SVGString,
			Fill:      drawReq.Fill,
			Stroke:    drawReq.Stroke,
			OpNum:     OpNum}

		OpNum++
		OpMutex.Unlock()

		// Disseminate Operation
		opBytes, _ := json.Marshal(op)
		opSig, _ := MinerInstance.PrivKey.Sign(rand.Reader, opBytes, nil)
		opSigStr := hex.EncodeToString(opSig)
		opInfo := blockchain.OperationInfo{
			AddSig: "",
			OpSig:  opSigStr,
			PubKey: pubKeyString,
			Op:     op}

		propOpArgs := PropagateOpArgs{
			OpInfo: opInfo,
			TTL:    TTL}

		log.Printf("write to ch")
		lmi.POpChan <- propOpArgs
		log.Printf("write to ch")
		lmi.SOpChan <- opInfo

		blockHash := ""
		count := 0

		// keep trying to validate the operation
		for {
			BlockCond.L.Lock()
			BlockCond.Wait()
			BlockCond.L.Unlock()

			// Check if it conflicts with the existing canvas
			err := ValidateOperation(op, pubKeyString, opSigStr)
			_, ok := err.(DuplicateError)
			if !ok {
				if err != nil {
					return err
				} else {
					// Keep count of how many times no duplicate.
					// If too many, reattempt operation
					count++
					if count > BLOCKS_BEFORE_REPROPAGATE {
						log.Printf("write to ch")
						lmi.POpChan <- propOpArgs
						log.Printf("write to ch")
						lmi.SOpChan <- opInfo
						fmt.Println("No dupe count too high - republishing")
						count = 0
					}

					fmt.Println("no duplicate yet - wait for new block")
					continue
				}
			}

			// Keep looping until there are NumValidate blocks
			blockHash := GetBlockHashOfShapeHash(opInfo.OpSig)
			if blockHash == "" {
				fmt.Println("Weird, no block hash - sleep then continue...")
				time.Sleep(1 * time.Second)
				continue
			}

			chain, _ := GetLongestPath(MinerInstance.Settings.GenesisBlockHash)
			numBlocksFollowing := 0
			for i := len(chain) - 1; i >= 0; i-- {
				blockByteData, _ := json.Marshal(chain[i])
				hashedBlock := utils.ComputeHash(blockByteData)
				hash := hex.EncodeToString(hashedBlock)
				if hash == blockHash {
					break
				} else {
					numBlocksFollowing++
				}
			}

			if numBlocksFollowing >= int(drawReq.ValidateNum) {
				break
			} else {
				fmt.Println("Not enough blocks to validate yet:", numBlocksFollowing)
			}
		}

		response.InkRemaining = uint32(CalculateInk(pubKeyString))
		response.ShapeHash = opInfo.OpSig
		response.BlockHash = blockHash
		return nil
	}
	err = fmt.Errorf("invalid user")
	return err

}

func (lmi *LibMinerInterface) Delete(req *libminer.Request, response *libminer.InkResponse) (err error) {
	if Verify(req.Msg, req.HashedMsg, req.R, req.S, MinerInstance.PrivKey) {
		var deleteReq libminer.DeleteRequest
		json.Unmarshal(req.Msg, &deleteReq)
		pubKeyString := utils.GetPublicKeyString(MinerInstance.PrivKey.PublicKey)
		fmt.Println("Delete called!")

		// Check if deletion is allowed
		path, _ := GetLongestPath(MinerInstance.Settings.GenesisBlockHash)
		err := MinerInstance.checkDeletion(deleteReq.ShapeHash, pubKeyString, path)
		if err != nil {
			return err
		}

		// Find the ADD Operation for metadata
		addBlockHash := GetBlockHashOfShapeHash(deleteReq.ShapeHash)
		if addBlockHash == "" {
			code := CheckStatusCode(libminer.ShapeOwnerError(deleteReq.ShapeHash))
			return errors.New(code)
		}

		addBlock := GetBlock(addBlockHash)
		var addOpInfo blockchain.OperationInfo
		for _, addInfo := range addBlock.OpHistory {
			if addInfo.OpSig == deleteReq.ShapeHash {
				addOpInfo = addInfo
				break
			}
		}

		if addOpInfo.Op.OpType != blockchain.ADD {
			code := CheckStatusCode(libminer.ShapeOwnerError(deleteReq.ShapeHash))
			return errors.New(code)
		}

		OpMutex.Lock()
		op := blockchain.Operation{
			OpType:    blockchain.DELETE,
			SVGString: addOpInfo.Op.SVGString,
			Fill:      addOpInfo.Op.Fill,
			Stroke:    addOpInfo.Op.Stroke,
			OpNum:     OpNum}

		OpNum++
		OpMutex.Unlock()

		// Disseminate Operation
		opBytes, _ := json.Marshal(op)
		opSig, _ := MinerInstance.PrivKey.Sign(rand.Reader, opBytes, nil)
		opInfo := blockchain.OperationInfo{
			AddSig: deleteReq.ShapeHash,
			OpSig:  hex.EncodeToString(opSig),
			PubKey: pubKeyString,
			Op:     op}

		propOpArgs := PropagateOpArgs{
			OpInfo: opInfo,
			TTL:    TTL}

		log.Printf("write to ch")
		lmi.POpChan <- propOpArgs
		log.Printf("write to ch")
		lmi.SOpChan <- opInfo

		count := 0

		fmt.Println("Delete ok - waiting now")

		// keep trying to validate the operation
		for {
			BlockCond.L.Lock()
			BlockCond.Wait()
			BlockCond.L.Unlock()

			// Keep looping until there are NumValidate blocks
			blockHash := GetBlockHashOfShapeHash(opInfo.OpSig)
			if blockHash == "" {
				fmt.Println("No del yet - sleep then continue...")
				count++

				if count > BLOCKS_BEFORE_REPROPAGATE {
					log.Printf("write to ch")
					lmi.POpChan <- propOpArgs
					log.Printf("write to ch")
					lmi.SOpChan <- opInfo
					count = 0
				}

				fmt.Println("no duplicate yet - wait for new block")
				continue
			}


			chain, _ := GetLongestPath(MinerInstance.Settings.GenesisBlockHash)
			numBlocksFollowing := 0
			for i := len(chain) - 1; i >= 0; i-- {
				blockByteData, _ := json.Marshal(chain[i])
				hashedBlock := utils.ComputeHash(blockByteData)
				hash := hex.EncodeToString(hashedBlock)
				if hash == blockHash {
					break
				} else {
					numBlocksFollowing++
				}
			}

			if numBlocksFollowing >= int(deleteReq.ValidateNum) {
				break
			} else {
				fmt.Println("Not enough blocks to validate yet:", numBlocksFollowing)
			}
		}

		response.InkRemaining = uint32(CalculateInk(pubKeyString))
		return nil
	}

	err = fmt.Errorf("invalid user")
	return err
}

func (lmi *LibMinerInterface) GetGenesisBlock(req *libminer.Request, response *string) (err error) {
	if Verify(req.Msg, req.HashedMsg, req.R, req.S, MinerInstance.PrivKey) {
		*response = MinerInstance.Settings.GenesisBlockHash
		return nil
	}
	err = fmt.Errorf("invalid user")
	return err
}

func (lmi *LibMinerInterface) GetChildren(req *libminer.Request, response *libminer.BlocksResponse) (err error) {
	if Verify(req.Msg, req.HashedMsg, req.R, req.S, MinerInstance.PrivKey) {
		var blockRequest libminer.BlockRequest
		json.Unmarshal(req.Msg, &blockRequest)
		if _, ok := ReadBlockChainMap(blockRequest.BlockHash); !ok {
			code := CheckStatusCode(libminer.InvalidBlockHashError(blockRequest.BlockHash))
			return errors.New(code)
		}
		children := GetBlockChildren(blockRequest.BlockHash)
		response.Blocks = children
		return nil
	}
	err = fmt.Errorf("invalid user")
	return err
}

func (lmi *LibMinerInterface) GetBlock(req *libminer.Request, response *libminer.BlocksResponse) (err error) {
	if Verify(req.Msg, req.HashedMsg, req.R, req.S, MinerInstance.PrivKey) {
		var blockRequest libminer.BlockRequest
		json.Unmarshal(req.Msg, &blockRequest)

		if blockIndex, ok := ReadBlockChainMap(blockRequest.BlockHash); ok {
			blockNode := BlockNodeArray[blockIndex]
			response.Blocks = []blockchain.Block{blockNode.Block}
			return nil
		}

		BlockArrayMutex.Lock()
		blockNodes := make([]blockchain.BlockNode, len(BlockNodeArray))
		copy(blockNodes, BlockNodeArray)
		BlockArrayMutex.Unlock()

		var children []blockchain.Block
		for _, bn := range(blockNodes) {
			if bn.Block.PrevHash == blockRequest.BlockHash {
				children = append(children, bn.Block)
			}
		}

		code := CheckStatusCode(libminer.InvalidBlockHashError(blockRequest.BlockHash))
		return errors.New(code)
	}

	err = fmt.Errorf("invalid user")
	return err
}

func (lmi *LibMinerInterface) GetOp(req *libminer.Request, response *libminer.OpResponse) (err error) {
	if Verify(req.Msg, req.HashedMsg, req.R, req.S, MinerInstance.PrivKey) {
		var opRequest libminer.OpRequest
		json.Unmarshal(req.Msg, &opRequest)

		blockHash := GetBlockHashOfShapeHash(opRequest.ShapeHash)
		if blockHash == "" {
			code := CheckStatusCode(libminer.InvalidShapeHashError(opRequest.ShapeHash))
			return errors.New(code)
		}

		blockIndex, _ := ReadBlockChainMap(blockHash)
		for _, opInfo := range BlockNodeArray[blockIndex].Block.OpHistory {
			if opInfo.OpSig == opRequest.ShapeHash {
				response.Op = opInfo.Op
				return nil
			}
		}

		code := CheckStatusCode(libminer.InvalidShapeHashError(opRequest.ShapeHash))
		return errors.New(code)
	}

	err = fmt.Errorf("invalid user")
	return err
}

/*******************************
| Blockchain functions
********************************/
// Appends the new block to BlockArray and updates BlockHashMap
func InsertBlock(newBlock blockchain.Block) (err error) {
	newBlockHash := GetBlockHash(newBlock)
	if _, ok := ReadBlockChainMap(newBlockHash); !ok && VerifyBlock(newBlock) {
		// Create a new BlockNode for newBlock and append it to BlockNodeArray
		fmt.Println("inserting:< Q", newBlock.PrevHash, ":", newBlock.Nonce)
		newPathInfo := LongestPathInfo{Len: 1, Path: []blockchain.Block{newBlock}}

		existingChildren, _ := ReadParentMap(newBlockHash)
		newBlockNode := blockchain.BlockNode{Block: newBlock, Children: existingChildren}

		newBlockIndex := WriteBlockNodeArray(newBlockNode)

		// Create an entry for newBlock in BlockHashMap
		WriteBlockChainMap(newBlockHash, newBlockIndex)

		// Get the path up until this current block and add the current block to the PathMap
		PathMapMutex.Lock() // Using a WriteLock since we're doing both R/W.
		if existingPath, hasPrevPath := PathMap[newBlockNode.Block.PrevHash]; hasPrevPath {
			// Has existing parent, need to add its path to the current block
			newPathInfo := LongestPathInfo{
				Len:  existingPath.Len + 1,
				Path: append(existingPath.Path, newBlock)}

			PathMap[newBlockHash] = newPathInfo
		} else {
			// Parent hasn't arrived yet. Path is itself.
			PathMap[newBlockHash] = newPathInfo
		}
		PathMapMutex.Unlock()

		// Check for any orphaned children that this newBlock is a parent to
		// and update the child's path in PathMap to include the newBlock's path + the childBlock
		if existingChildren, hasExistingChildren := ReadParentMap(GetBlockHash(newBlock)); hasExistingChildren {
			for _, existingChildIndex := range existingChildren {
				existingChildBlock := BlockNodeArray[existingChildIndex].Block
				existingChildHash := GetBlockHash(existingChildBlock)
				childPathInfo := LongestPathInfo{Len: newPathInfo.Len + 1, Path: append(newPathInfo.Path, existingChildBlock)}
				WritePathMap(existingChildHash, childPathInfo)
			}
		}

		// Update the entry for newBlock's parent in BlockNodeArray
		// If the parent exists in the blockchain, simply append this new block as a child of the parent
		// If the parent does not exist either because:
		// 		1) It is an invalid block
		//			- Adding this to the BlockNodeArray will make this an unreachable Node
		//      2) The parent has yet to arrive
		//			- When the parent arrives, it will append all the pending children in ParentHashMap
		if parentIndex, ok := ReadBlockChainMap(newBlock.PrevHash); ok {
			BlockArrayMutex.Lock()
			parentBlockNode := &BlockNodeArray[parentIndex]
			parentBlockNode.Children = append(parentBlockNode.Children, newBlockIndex)
			BlockArrayMutex.Unlock()
		} else {
			ParentMapMutex.Lock()
			existingChildren, _ := ParentHashMap[newBlock.PrevHash]
			ParentHashMap[newBlock.PrevHash] = append(existingChildren, newBlockIndex)
			ParentMapMutex.Unlock()
		}

		BlockCond.L.Lock()
		BlockCond.Broadcast()
		BlockCond.L.Unlock()

		//fmt.Println("parent's node with new child:", parentBlockNode)
		return nil
	}
	err = fmt.Errorf("Block hash does not match up with block contents!")
	return err
}

// Do we need this?
// It seems like the only block individually retrieved is the GenesisBlock
func GetBlock(blockHash string) blockchain.Block {
	index, _ := ReadBlockChainMap(blockHash)
	return BlockNodeArray[index].Block
}

func GetBlockChildren(blockHash string) []blockchain.Block {
	var children []blockchain.Block
	parentIndex, _ := ReadBlockChainMap(blockHash)
	for _, childIndex := range BlockNodeArray[parentIndex].Children {
		children = append(children, BlockNodeArray[childIndex].Block)
	}
	return children
}

func VerifyBlock(block blockchain.Block) bool {
	hash := GetBlockHash(block)
	if len(block.OpHistory) == 0 {
		return pow.Verify(hash, int(MinerInstance.Settings.PoWDifficultyNoOpBlock))
	}
	return pow.Verify(hash, int(MinerInstance.Settings.PoWDifficultyOpBlock))
}

// Returns an array of Blocks of the longest path that follow initBlockHash and length of the longest path
func GetLongestPath(initBlockHash string) ([]blockchain.Block, int) {
	//fmt.Println("running get longest path with block hash: ", initBlockHash)
	defer Recover()
	blockChain := make([]blockchain.Block, 0)

	initBIndex, blockExists := ReadBlockChainMap(initBlockHash)

	if !blockExists {
		return blockChain, 0
	}

	blockChain = append(blockChain, BlockNodeArray[initBIndex].Block)

	// For the genesis block, we can return the entire length of the continuous blockchain since it is cached in PathMap
	if initBlockHash == MinerInstance.Settings.GenesisBlockHash {
		PathMapMutex.RLock()
		defer PathMapMutex.RUnlock()
		var maxHash string
		var maxPathInfo LongestPathInfo
		for bHash, pathInfo := range PathMap {
			if pathInfo.Len > maxPathInfo.Len {
				maxHash = bHash
				maxPathInfo = pathInfo
			} else if pathInfo.Len == maxPathInfo.Len {
				// Break the tie by comparing the hash
				if strings.Compare(bHash, maxHash) > 0 {
					maxHash = bHash
					maxPathInfo = pathInfo
				}
			}
		}

		return maxPathInfo.Path, maxPathInfo.Len
	}

	// If it isn't the Genesis Block, we only return the subset starting from initBIndex

	// If there's no children, return the current block
	if len(BlockNodeArray[initBIndex].Children) == 0 {
		return blockChain, 1
	}

	var longestPath []blockchain.Block
	maxLen := -1

	for _, childIndex := range BlockNodeArray[initBIndex].Children {
		// TODO remove
		blenn := len(BlockNodeArray)
		if childIndex >= blenn {
			log.Println("REAL BLOCKNODE param: %+v, blenn: %d", BlockNodeArray[initBIndex], blenn)
		}

		child := BlockNodeArray[childIndex]

		childHash := GetBlockHash(child.Block)
		childPath, childLen := GetLongestPath(childHash)
		longestPathBlockHash := ""

		// If the childLen is equal to the max length, we use their hashes to determine which path to build off of
		// If childLen > maxLen, we simply update the longestPath
		if (maxLen == childLen && strings.Compare(childHash, longestPathBlockHash) > 0) || maxLen < childLen {
			longestPath = childPath
			maxLen = childLen
			longestPathBlockHash = childHash
		}
	}

	blockChain = append(blockChain, longestPath...)
	return blockChain, maxLen + 1
}

/////////////////////////////////////////////////////////////////////////////////////////////////////
/*********** HELPERS TO ALLOW FOR CONCURRENT ACCESS TO MUTABLE GLOBAL VARS *************************/
// Returns the index of the Block with k Blockhash
func ReadBlockChainMap(k string) (blockChainIndex int, exists bool) {
	BlockChainMutex.RLock()
	defer BlockChainMutex.RUnlock()
	if v, ok := BlockHashMap[k]; ok {
		return v, true
	}

	// fmt.Printf("Error, not blockchain map, Key: %s\n", k)
	// fmt.Printf("PRINTING BLOCKCHAIN MAP - Len: %d \n %+v\n\n", len(BlockHashMap), BlockHashMap)
	return -1, false
}

func WriteBlockChainMap(k string, v int) {
	BlockChainMutex.Lock()
	defer BlockChainMutex.Unlock()
	BlockHashMap[k] = v
}

// Returns the position the BlockNode was inserted in
func WriteBlockNodeArray(b blockchain.BlockNode) int {
	BlockArrayMutex.Lock()
	defer BlockArrayMutex.Unlock()

	// Check if this block has already been added to the array
	// since we're duplicating lots of blocks
	i, alreadyAdded := ReadBlockChainMap(GetBlockHash(b.Block))
	if !alreadyAdded {
		BlockNodeArray = append(BlockNodeArray, b)
		return len(BlockNodeArray) - 1
	}

	fmt.Println("Already added: %s", GetBlockHash(b.Block))
	return i
}

func WriteParentMap(k string, v []int) {
	ParentMapMutex.Lock()
	defer ParentMapMutex.Unlock()
	ParentHashMap[k] = v
}

func ReadParentMap(k string) (childIndices []int, hasChildren bool) {
	ParentMapMutex.RLock()
	defer ParentMapMutex.RUnlock()
	if v, ok := ParentHashMap[k]; ok {
		return v, true
	}

	return []int{}, false
}

func WritePathMap(k string, v LongestPathInfo) {
	PathMapMutex.Lock()
	defer PathMapMutex.Unlock()

	PathMap[k] = v
}

// Get's the given BlockHash's longest path
func ReadPathMap(k string) (pathInfo LongestPathInfo, hasChildren bool) {
	PathMapMutex.RLock()
	defer PathMapMutex.RUnlock()

	pathInfo, hasChildren = PathMap[k]
	return pathInfo, hasChildren
}

/***********END OF HELPERS TO ALLOW FOR CONCURRENT ACCESS TO MUTABLE GLOBAL VARS **********************/
////////////////////////////////////////////////////////////////////////////////////////////////////////

// Returns an array of Blocks that are on the same path, ahead of the hash
func GetPath(targetBlockHash string) []blockchain.Block {
	// TODO remove
	// lastIndex, _ := ReadBlockChainMap(targetBlockHash)
	// lastBlock := blockNodeArray[lastIndex].Block
	// blockChain := []blockchain.Block{lastBlock}
	// for {
	// 	if _, ok := ReadBlockChainMap(lastBlock.PrevHash); !ok || lastBlock.PrevHash == MinerInstance.Settings.GenesisBlockHash {
	// 		return blockChain, nil
	// 	}

	// 	lastIndex, _ = ReadBlockChainMap(lastBlock.PrevHash)
	// 	lastBlock = blockNodeArray[lastIndex].Block
	// 	blockChain = append([]blockchain.Block{lastBlock}, blockChain...)
	// }

	pathInfo, _ := ReadPathMap(targetBlockHash)
	return pathInfo.Path
}

// Calculates how much ink a particular miner public key has
func CalculateInk(minerKey string) int {
	blockChain, _ := GetLongestPath(MinerInstance.Settings.GenesisBlockHash)
	var inkAmt uint32
	for _, block := range blockChain {
		if block.MinerPubKey == minerKey {
			if len(block.OpHistory) == 0 {
				inkAmt += MinerInstance.Settings.InkPerNoOpBlock
			} else {
				inkAmt += MinerInstance.Settings.InkPerOpBlock
			}
		}

		for _, opInfo := range block.OpHistory {
			op := opInfo.Op
			if opInfo.PubKey == minerKey {
				shape, err := MinerInstance.getShapeFromOp(op)
				if err != nil {
					fmt.Println("CRITICAL ERROR: BAD SHAPE IN BLOCKCHAIN")
					continue
				}

				_, cost := shape.SubArrayAndCost()
				if op.OpType == blockchain.ADD {
					inkAmt -= uint32(cost)
				} else {
					inkAmt += uint32(cost)
				}
			}
		}
	}
	fmt.Println("this miner has this much ink:", int(inkAmt))
	return int(inkAmt)
}

/*******************************
| Server Management functions
********************************/

func (msi *MinerServerInterface) Register(minerAddr net.Addr) {
	reqArgs := minerserver.MinerInfo{Address: minerAddr, Key: MinerInstance.PrivKey.PublicKey}
	var resp minerserver.MinerNetSettings
	err := msi.Client.Call("RServer.Register", reqArgs, &resp)
	resp.PoWDifficultyOpBlock ++
	resp.PoWDifficultyNoOpBlock ++
	CheckError(err, "Register:Client.Call")
	MinerInstance.Settings = resp
}

func (msi *MinerServerInterface) ServerHeartBeat() {
	var ignored bool
	//fmt.Println("ServerHeartBeat::Sending heartbeat")
	err := msi.Client.Call("RServer.HeartBeat", MinerInstance.PrivKey.PublicKey, &ignored)
	if CheckError(err, "ServerHeartBeat") {
		//Reconnect to server if timed out
		msi.Register(MinerInstance.Addr)
	}
}

func (msi *MinerServerInterface) GetPeers(addrSet []net.Addr) {
	var blockchainResp []blockchain.Block
	for _, addr := range addrSet {
		if _, ok := PeerList[addr.String()]; !ok {
			fmt.Println("GetPeers::Connecting to address: ", addr.String())
			LocalAddr, err := net.ResolveTCPAddr("tcp", ":0")
			if CheckError(err, "GetPeers:ResolvePeerAddr") {
				continue
			}

			PeerAddr, err := net.ResolveTCPAddr("tcp", addr.String())
			if CheckError(err, "GetPeers:ResolveLocalAddr") {
				continue
			}

			conn, err := net.DialTCP("tcp", LocalAddr, PeerAddr)
			if CheckError(err, "GetPeers:DialTCP") {
				continue
			}

			client := rpc.NewClient(conn)

			args := ConnectArgs{MinerInstance.Addr}
			err = client.Call("Peer.Connect", args, &blockchainResp)
			if CheckError(err, "GetPeers:Connect") {
				continue
			}
			for _, block := range blockchainResp {
				InsertBlock(block)
			}
			PeerList[addr.String()] = &Peer{client, time.Now()}
		}
	}
}

/*******************************
| Connection Management
********************************/
// This function has 5 purposes:
// 1. Send the server heartbeat to maintain connectivity
// 2. Send miner heartbeats to maintain connectivity with peers
// 3. Check for stale peers and remove them from the list
// 4. Request new nodes from server and connect to them when peers drop too low
// 5. When a operation or block is sent through the channel, heartbeat will be replaced by Propagate<Type>
// This is the central point of control for the peer connectivity

func ManageConnections(pop chan PropagateOpArgs, pblock chan PropagateBlockArgs, peerconn chan net.Addr) {
	// Send heartbeats at three times the timeout interval to be safe
	interval := time.Duration(MinerInstance.Settings.HeartBeat / 5)
	heartbeat := time.Tick(interval * time.Millisecond)
	count := 0
	for {
		select {
		case <-heartbeat:
			MinerInstance.MSI.ServerHeartBeat()
			if count >= 50 {
				PeerSync()
				count = 0
			} else {
				count++
				PeerHeartBeats()
			}
		case addr := <-peerconn:
			// Connection request from peerRpc
			addrSet := []net.Addr{addr}
			MinerInstance.MSI.GetPeers(addrSet)
		case op := <-pop:
			MinerInstance.MSI.ServerHeartBeat()
			PeerPropagateOp(op)
		case block := <-pblock:
			MinerInstance.MSI.ServerHeartBeat()
			PeerPropagateBlock(block)
		default:
			CheckLiveliness()
			if len(PeerList) < int(MinerInstance.Settings.MinNumMinerConnections) {
				var addrSet []net.Addr
				MinerInstance.MSI.Client.Call("RServer.GetNodes", MinerInstance.PrivKey.PublicKey, &addrSet)
				MinerInstance.MSI.GetPeers(addrSet)
			}
		}
	}
}

// Try to sync up with peers once in a while
func PeerSync() {
	fmt.Println("Performing a sync")
	for addr, peer := range PeerList {
		var blockchainResp []blockchain.Block
		empty := new(Empty)
		err := peer.Client.Call("Peer.GetBlockChain", empty, &blockchainResp)
		if !CheckError(err, "PeerSync:"+addr) {
			peer.LastHeartBeat = time.Now()
			for _, block := range blockchainResp {
				InsertBlock(block)
			}
		}
	}
}

// Send a heartbeat call to each peer
func PeerHeartBeats() {
	for addr, peer := range PeerList {
		empty := new(Empty)
		err := peer.Client.Call("Peer.Hb", &empty, &empty)
		if !CheckError(err, "PeerHeartBeats:"+addr) {
			peer.LastHeartBeat = time.Now()
		}
	}
}

// Send a PropagateOp call to each peer
// Assumption: Nothing needs to be done on the miner itself, only send the op onwards
func PeerPropagateOp(op PropagateOpArgs) {
	for _, peer := range PeerList {
		empty := new(Empty)
		args := PropagateOpArgs{op.OpInfo, op.TTL}
		peer.Client.Call("Peer.PropagateOp", args, &empty)
	}
}

// Send a PropagateBlock call to each peer
// Assumption: Nothing needs to be done on the miner itself, only send the block onwards
func PeerPropagateBlock(block PropagateBlockArgs) {
	for _, peer := range PeerList {
		empty := new(Empty)
		args := PropagateBlockArgs{block.Block, block.TTL}
		peer.Client.Call("Peer.PropagateBlock", args, &empty)
	}
}

// Look through current active connections and delete them if they are not live
func CheckLiveliness() {
	interval := time.Duration(MinerInstance.Settings.HeartBeat) * time.Millisecond
	for addr, peer := range PeerList {
		if time.Since(peer.LastHeartBeat) > interval {
			fmt.Println("Stale connection: ", addr, " deleting")
			peer.Client.Close()
			delete(PeerList, addr)
		}
	}
}

/*******************************
| Crypto-Management
********************************/
// The problemsolver handles 4 main functions
// 1. Spins new workers for a new job
// 2. Kills old workers for a new job
// 3. Receive job updates via the given channels
// 4. TODO: Return solution

func ProblemSolver(sop chan blockchain.OperationInfo, sblock chan blockchain.Block, pblock chan PropagateBlockArgs) {
	// Channel for receiving the final block w/ nonce from workers
	solved := make(chan blockchain.Block)
	workingSet := make([]blockchain.OperationInfo, 0)

	// Channel returned by a job call that can kill the workers for that particular job
	var done chan bool

	for {
		select {
		case op := <-sop:
			// Received an op from somewhere
			// Assuming it is properly validated
			// Add it to the block we were working on
			// reissue job
			fmt.Println("got new op to hash")
			// Kill current job
			close(done)
			close(solved)

			// Make a new channel
			solved = make(chan blockchain.Block)

			workingSet = append(workingSet, op)

			chain, chainLen := GetLongestPath(MinerInstance.Settings.GenesisBlockHash)
			workingSet = ValidateOps(workingSet, chain)

			if len(workingSet) == 0 {
				done = NoopJob(GetBlockHash(chain[chainLen-1]), solved)
			} else {
				done = OpJob(GetBlockHash(chain[chainLen-1]), workingSet, solved)
			}

		case block := <-sblock:
			// Received a block from somewhere
			// Assume that this block was validated
			// Assume this is the next block to build off of
			// Reissue a job with this blockhash as prevBlock
			fmt.Println("got new block to hash")

			// Kill current job
			close(done)
			close(solved)

			// Make a new channel
			solved = make(chan blockchain.Block)

			// Assume this was block was validated
			// Assume this block has already been inserted
			chain, _ := GetLongestPath(MinerInstance.Settings.GenesisBlockHash)
			workingSet = ValidateOps(workingSet, chain)
			if len(workingSet) == 0 {
				done = NoopJob(GetBlockHash(block), solved)
			} else {
				done = OpJob(GetBlockHash(block), workingSet, solved)
			}
		case sol := <-solved:
			if len(sol.OpHistory) > 0 {
				fmt.Println("got a solution", sol.OpHistory[0])
			}

			// Kill current job
			close(done)
			close(solved)
			// Make a new channel
			solved = make(chan blockchain.Block)

			// Insert block into our data structure
			InsertBlock(sol)
			pblock <- PropagateBlockArgs{sol, TTL}

			//fmt.Println("inserted solution: ", BlockNodeArray)
			// Start a job on the longest block in the chain
			chain, chainLen := GetLongestPath(MinerInstance.Settings.GenesisBlockHash)
			//fmt.Println("state of the longest blockchain", blockchain)
			lastblock := chain[chainLen-1]
			done = NoopJob(GetBlockHash(lastblock), solved)

			PrintBlockChain(chain)
		default:
			if CurrJobId == 0 {
				fmt.Println("Initiating the first job")
				done = NoopJob(MinerInstance.Settings.GenesisBlockHash, solved)
			}
		}
	}
}

// Initiate a job with an empty op array and a blockhash
func NoopJob(hash string, solved chan blockchain.Block) chan bool {
	CurrJobId++
	fmt.Println("Starting job:", CurrJobId)
	block := blockchain.Block{PrevHash: hash,
		MinerPubKey: utils.GetPublicKeyString(MinerInstance.PrivKey.PublicKey)}
	done := make(chan bool)
	for i := 0; i <= MAX_THREADS; i++ {
		CurrJobId++
		// Split up the start by the maximum number of threads we allow
		start := math.MaxUint32 / MAX_THREADS * i
		go pow.Solve(block, MinerInstance.Settings.PoWDifficultyNoOpBlock, uint32(start), solved, done)
	}
	return done
}

// Initiate a job with a predefined op array
func OpJob(hash string, Ops []blockchain.OperationInfo, solved chan blockchain.Block) chan bool {
	CurrJobId++
	fmt.Println("Starting job:", CurrJobId)
	block := blockchain.Block{PrevHash: hash,
		OpHistory:   Ops,
		MinerPubKey: utils.GetPublicKeyString(MinerInstance.PrivKey.PublicKey)}
	done := make(chan bool)
	for i := 0; i <= MAX_THREADS; i++ {
		CurrJobId++
		// Split up the start by the maximum number of threads we allow
		start := math.MaxUint32 / MAX_THREADS * i
		go pow.Solve(block, MinerInstance.Settings.PoWDifficultyOpBlock, uint32(start), solved, done)
	}
	return done
}

/*******************************
| Helpers
********************************/
func Verify(msg []byte, sign []byte, R, S big.Int, privKey *ecdsa.PrivateKey) bool {
	h := md5.New()
	h.Write(msg)
	hash := hex.EncodeToString(h.Sum(nil))
	if hash == hex.EncodeToString(sign) && ecdsa.Verify(&privKey.PublicKey, sign, &R, &S) {
		return true
	} else {
		fmt.Println("invalid access\n")
		return false
	}
}
func CheckError(err error, parent string) bool {
	if err != nil {
		fmt.Println(parent, ":: found error! ", err)
		return true
	}
	return false
}

func CheckStatusCode(err error) string {
	switch err.(type) {
	case libminer.ShapeSvgStringTooLongError:
		return "1" + " " + err.Error()
	case libminer.InvalidShapeSvgStringError:
		return "2" + " " + err.Error()
	case libminer.InsufficientInkError:
		return "3" + " " + err.Error()
	case libminer.ShapeOverlapError:
		return "4" + " " + err.Error()
	case libminer.OutOfBoundsError:
		return "5" + " " + err.Error()
	case libminer.InvalidBlockHashError:
		return "6" + " " + err.Error()
	case libminer.ShapeOwnerError:
		return "7" + " " + err.Error()
	case libminer.InvalidShapeHashError:
		return "8" + " " + err.Error()
	default:
		return "9"
	}
}

func ExtractKeyPairs(pubKey, privKey string) {
	var PublicKey *ecdsa.PublicKey
	var PrivateKey *ecdsa.PrivateKey

	pubKeyBytesRestored, _ := hex.DecodeString(pubKey)
	privKeyBytesRestored, _ := hex.DecodeString(privKey)

	pub, err := x509.ParsePKIXPublicKey(pubKeyBytesRestored)
	CheckError(err, "ExtractKeyPairs:ParsePKIXPublicKey")
	PublicKey = pub.(*ecdsa.PublicKey)

	PrivateKey, err = x509.ParseECPrivateKey(privKeyBytesRestored)
	CheckError(err, "ExtractKeyPairs:ParseECPrivateKey")

	r, s, _ := ecdsa.Sign(rand.Reader, PrivateKey, []byte("data"))

	if !ecdsa.Verify(PublicKey, []byte("data"), r, s) {
		fmt.Println("ExtractKeyPairs:: Key pair incorrect, please recheck")
	}
	MinerInstance.PrivKey = PrivateKey
	fmt.Println("ExtractKeyPairs:: Key pair verified")
}

func pubKeyToString(key ecdsa.PublicKey) string {
	return string(elliptic.Marshal(key.Curve, key.X, key.Y))
}

func GetBlockHash(block blockchain.Block) string {
	h := md5.New()
	bytes, _ := json.Marshal(block)
	h.Write(bytes)
	hash := hex.EncodeToString(h.Sum(nil))
	return hash
}

// Checks if this operation has already been incorporated in the longest path of the blockchain
// If it is in the blockchain, return the block where the operation is in
func GetBlockHashOfShapeHash(opSig string) string {
	blockchain, _ := GetLongestPath(MinerInstance.Settings.GenesisBlockHash)

	for _, block := range blockchain {
		for _, op := range block.OpHistory {
			if op.OpSig == opSig {
				blockByteData, _ := json.Marshal(block)
				hashedBlock := utils.ComputeHash(blockByteData)
				return hex.EncodeToString(hashedBlock)
			}
		}
	}

	return ""
}

func PrintBlockChain(blocks []blockchain.Block) {
	fmt.Println("Current amount of blocks we have: ", len(BlockHashMap))
	for i, block := range blocks {
		if i != 0 {
			if len(block.PrevHash) < 6 || len(block.MinerPubKey) < 6 {
				continue
			}
			fmt.Print("<- ", block.PrevHash[0:5], ":", block.MinerPubKey[len(block.MinerPubKey)-5:], ":")
			for _, opinfo := range block.OpHistory {
				if opinfo.Op.OpType == blockchain.ADD {
					fmt.Print("-ADD:", opinfo.Op.SVGString, ":", opinfo.OpSig,"-")
				} else {
					fmt.Print("-DELETE:", opinfo.Op.SVGString, ":", opinfo.OpSig,"-")
				}
			}
			fmt.Print(" ->\n")
		} else {
			fmt.Println("<- ", MinerInstance.Settings.GenesisBlockHash, " ->")
		}
	}
	fmt.Println("Length of the blockchain: ", len(blocks))
}

func RecoverTemp() {
	p, l := GetLongestPath(MinerInstance.Settings.GenesisBlockHash)
	fmt.Printf("Len of Blockchain Path is: %d. Path:\n", l)

	for i, pp := range p {
		fmt.Printf("B%d : BlockHash: %s PrevHash: %s\n", i, GetBlockHash(pp), pp.PrevHash)
	}

	//fmt.Println("")

	// if len(blocks) >= 15 {
	// 	defer RecoverTemp()
	// 	os.Exit(1)
	// }
	// for _, block := range blocks {
	// 	fmt.Print("<- ", block.PrevHash, ":", block.Nonce, "->")
	// }
	// fmt.Print("\n")
}

func Recover() {
	// recover from panic caused by writing to a closed channel
	if r := recover(); r != nil {
		fmt.Println("recovered from GetLongestPath")
		blockhash, _ := json.Marshal(BlockHashMap)
		blockarray, _ := json.Marshal(BlockNodeArray)
		ioutil.WriteFile("./output/blockhashmap.txt", blockhash, 0644)
		ioutil.WriteFile("./output/blockhasharray.txt", blockarray, 0644)
		return
	}
}

// Code from https://gist.github.com/jniltinho/9787946
func GeneratePublicIP() string {
	addrs, err := net.InterfaceAddrs()
	CheckError(err, "GeneratePublicIP")

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String() + ":"
			}
		}
	}

	return "Could not find IP"
}

/*******************************
| Main
********************************/
func Mine(serverIP, pubKey, privKey string) {
	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})
	// serverIP, pubKey, privKey := os.Args[1], os.Args[2], os.Args[3]
	serverIP = os.Args[1]

	BlockCond = &sync.Cond{L: &sync.Mutex{}}

	// 1. Setup the singleton miner instance
	MinerInstance = new(Miner)
	// Extract key pairs
	ExtractKeyPairs(pubKey, privKey)
	// Listening Address
	publicIP := GeneratePublicIP()
	fmt.Println(publicIP)

	ln, _ := net.Listen("tcp", publicIP)
	addr := ln.Addr()
	MinerInstance.Addr = addr

	// 2. Create communication channels between goroutines
	pop := make(chan PropagateOpArgs, 1024)
	pblock := make(chan PropagateBlockArgs, 1024)
	sop := make(chan blockchain.OperationInfo, 1024)
	sblock := make(chan blockchain.Block, 1024)
	peerconn := make(chan net.Addr, 64)

	// 3. Setup Miner-Miner Listener
	go listenPeerRpc(ln, MinerInstance, pop, pblock, sop, sblock, peerconn)

	// Connect to Server
	MinerInstance.ConnectToServer(serverIP)
	MinerInstance.MSI.Register(addr)

	// Initialize mutexes for concurrent R/W of BlockChain global variables
	OpMutex = &sync.Mutex{}
	BlockChainMutex = &sync.RWMutex{}
	BlockArrayMutex = &sync.Mutex{}
	ParentMapMutex = &sync.RWMutex{}
	PathMapMutex = &sync.RWMutex{}

	// Initialize the hash map, block node array, and path map with the genesis block
	BlockHashMap[MinerInstance.Settings.GenesisBlockHash] = 0
	WriteBlockNodeArray(blockchain.BlockNode{})
	dummyGenesisBlock := blockchain.Block{}
	WritePathMap(MinerInstance.Settings.GenesisBlockHash, LongestPathInfo{Len: 1, Path: []blockchain.Block{dummyGenesisBlock}})

	// 4. Setup Miner Heartbeat Manager
	go ManageConnections(pop, pblock, peerconn)

	// 5. Setup Problem Solving
	go ProblemSolver(sop, sblock, pblock)

	// 6. Setup Client-Miner Listener (this thread)
	OpenLibMinerConn(":0", pop, sop)
}
