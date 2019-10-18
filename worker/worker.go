package worker

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

// GoldenNonce computed from nonce appended to input string
type GoldenNonce struct {
	Nonce uint32
	Hash  string
}

// CalculateGoldenNonce computes golden nonce for the string concatenated with all nonces in range [start, end)
func CalculateGoldenNonce(contents string, start uint32, end uint32, target int) (*GoldenNonce, error) {
	for i := start; i < end; i++ {
		hash, err := hash(contents, i)
		if err != nil {
			return nil, err
		}

		zeros := leadingZeros(hash)
		if zeros >= target {
			return &GoldenNonce{i, hex.EncodeToString(hash)}, nil
		}
	}
	return nil, errors.New("Unable to find nonce in range")
}

func hash(block string, nonce uint32) ([]byte, error) {
	// Convert the nonce to a byte[]
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, nonce)
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
		for i := 7; i >= 0; i-- {
			mask := byte(1 << uint(i))
			if b&mask != 0 {
				return leadingZeros
			}
			leadingZeros++
		}
	}
	return leadingZeros
}
