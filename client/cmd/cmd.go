package cmd

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
)

const (
	noncesProcessedPerSecond uint32 = 470000
)

// WorkerConfig built from Command Line
type WorkerConfig struct {
	Block        *string
	LeadingZeros int
	Workers      int
	Timeout      int
	Confidence   int
	UseECS       bool
}

// LogConfig will output the configuration being used
func (wc *WorkerConfig) LogConfig() {
	strategy := "Docker"
	if wc.UseECS {
		strategy = "ECS"
	}

	log.Printf("--- Configuration ---")
	log.Printf("Block: %s", *wc.Block)
	log.Printf("Timeout: %d seconds", wc.Timeout)
	log.Printf("Leading zeros: %d", wc.LeadingZeros)
	log.Printf("Workers: %d", wc.Workers)
	log.Printf("Deployment strategy: %s", strategy)
	log.Printf("---------------------")
}

// ParseArgs will parse the command line arguments and produce a configuration
func ParseArgs() (*WorkerConfig, error) {
	// Two modes
	directCommand := flag.NewFlagSet("direct", flag.ExitOnError)
	indirectCommand := flag.NewFlagSet("indirect", flag.ExitOnError)

	// Direct mode args
	directBlock := directCommand.String("block", "COMSM0010cloud", "block of data the nonce is appended to")
	directLeadingZeros := directCommand.Int("d", 20, "number of leading zeros")
	directTimeout := directCommand.Int("timeout", 360, "timeout in seconds")
	directWorkers := directCommand.Int("n", 1, "number of workers")
	directECS := directCommand.Bool("use-ecs", false, "use ecs as a task scheduler")

	// Indirect mode args
	indirectBlock := indirectCommand.String("block", "COMSM0010cloud", "block of data the nonce is appended to")
	indirectLeadingZeros := indirectCommand.Int("d", 20, "number of leading zeros")
	indirectTimeout := indirectCommand.Int("timeout", 360, "timeout in seconds")
	indirectConfidence := indirectCommand.Int("confidence", 95, "confidence in finding the result, as a percentage")
	indirectECS := indirectCommand.Bool("use-ecs", false, "use ecs as a task scheduler")

	if len(os.Args) < 2 {
		fmt.Println("direct or indirect subcommand is required")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "direct":
		directCommand.Parse(os.Args[2:])
	case "indirect":
		indirectCommand.Parse(os.Args[2:])
	default:
		fmt.Println("[direct] mode")
		directCommand.PrintDefaults()
		fmt.Println("\n[indirect] mode")
		indirectCommand.PrintDefaults()
		os.Exit(1)
	}

	// Validate results
	if directCommand.Parsed() {
		if len(*directBlock) == 0 {
			return nil, errors.New("Invalid data block, must be non empty")
		} else if *directLeadingZeros <= 0 {
			return nil, errors.New("Invalid leading zeros, must be greater than 0")
		} else if *directTimeout <= 0 {
			return nil, errors.New("Invalid timeout, must be greater than 0")
		} else if *directWorkers <= 0 || *directWorkers >= 32 {
			return nil, errors.New("Invalid number of workers, must be in range [0, 32)")
		}

		return &WorkerConfig{
			Block:        directBlock,
			LeadingZeros: *directLeadingZeros,
			Timeout:      *directTimeout,
			Workers:      *directWorkers,
			Confidence:   100,
			UseECS:       *directECS,
		}, nil
	}

	if indirectCommand.Parsed() {
		if len(*indirectBlock) == 0 {
			return nil, errors.New("Invalid data block, must be non empty")
		} else if *indirectLeadingZeros <= 0 {
			return nil, errors.New("Invalid leading zeros, must be greater than 0")
		} else if *indirectTimeout <= 0 {
			return nil, errors.New("Invalid timeout, must be greater than 0")
		} else if *indirectConfidence <= 0 || *indirectConfidence > 100 {
			return nil, errors.New("Invalid number of workers, must be in range [0, 100]")
		}

		workers := calculateWorkers(*indirectTimeout, *indirectConfidence)
		if workers >= 32 {
			return nil, errors.New("Unable to satisfy constraints without using more than 32 workers")
		}

		return &WorkerConfig{
			Block:        indirectBlock,
			LeadingZeros: *indirectLeadingZeros,
			Timeout:      *indirectTimeout,
			Confidence:   *indirectConfidence,
			Workers:      workers,
			UseECS:       *indirectECS,
		}, nil
	}

	return nil, errors.New("Unable to parse CLI args")
}

func calculateWorkers(timeout int, confidence int) int {
	totalNumbersToSearch := ^uint32(0)
	numberOfSecondsNeeded := float64(totalNumbersToSearch) / float64(noncesProcessedPerSecond)
	workersNeededToFullySearch := numberOfSecondsNeeded / float64(timeout)
	workersNeededForConfidence := math.Ceil(workersNeededToFullySearch * float64(confidence) * 0.01)
	return int(workersNeededForConfidence)
}
