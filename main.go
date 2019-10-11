package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
)

func hash(block string, nonce uint32) ([]byte, error) {
	// Convert the nonce to a byte[]
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, nonce)
	if err != nil {
		return nil, err
	}

	blockBytes := []byte(block)
	bytes := append(blockBytes, buf.Bytes()...)

	// Complete one or two hashes
	firstHash := sha256.Sum256(bytes)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash[:], nil
}

func leadingZeros(arr []byte) int {
	leadingZeros := 0

	for _, b := range arr {
		for i := 0; i < 8; i++ {
			mask := byte(1 << uint(i))
			if b&mask != 0 {
				return leadingZeros
			}
			leadingZeros++
		}
	}
	return leadingZeros
}

func main() {
	for i := uint32(0); i < ^uint32(0); i++ {
		hash, err := hash("COMSM0010cloud", i)
		if err != nil {
			fmt.Printf("Something went wrong: %s", err.Error())
			os.Exit(1)
			return
		}

		zeros := leadingZeros(hash)
		if zeros > 8 {
			fmt.Printf("The golden nonce is: %d, with hash: %s\n", i, hex.EncodeToString(hash))
			os.Exit(0)
			return
		}
	}
}
