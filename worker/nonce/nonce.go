package nonce

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// GoldenNonce computed from nonce appended to input string
type GoldenNonce struct {
	Nonce uint32
	Hash  string
}

// WorkerConfig provides the necessary parameters to compute a golden nonce
type WorkerConfig struct {
	Contents   string
	LowerBound uint32
	UpperBound uint32
	Target     int
	DebugDesc  string
}

// NoNonceFoundError is thrown when a nonce cannot be found
type NoNonceFoundError struct {
	err string
}

// var (
// 	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
// 		Name: "worker_processed_ops_total",
// 		Help: "The total number of processed nonces",
// 	})
// )

func (e *NoNonceFoundError) Error() string {
	return fmt.Sprintf("Couldn't find nonce: %s", e.err)
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

// CalculateGoldenNonce computes golden nonce for the string concatenated with all nonces in range [start, end)
func CalculateGoldenNonce(config *WorkerConfig) (*GoldenNonce, error) {
	for i := config.LowerBound; i < config.UpperBound; i++ {
		hash, err := hash(config.Contents, i)
		if err != nil {
			return nil, err
		}

		zeros := leadingZeros(hash)
		// go func() {
		// 	opsProcessed.Inc()
		// }()
		if zeros >= config.Target {
			return &GoldenNonce{i, hex.EncodeToString(hash)}, nil
		}
	}
	return nil, &NoNonceFoundError{fmt.Sprintf("No nonce found of length %d between %d and %d", config.Target, config.LowerBound, config.UpperBound)}
}
