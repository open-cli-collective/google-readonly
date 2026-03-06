// Package bulk provides shared ID resolution and result types for bulk operations.
package bulk

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config specifies how to resolve IDs for a bulk operation.
type Config struct {
	Args  []string // positional args
	Stdin bool     // --stdin flag
	Query string   // --query value
}

// ResolveIDs returns IDs from exactly one input source.
// queryFn is called when Query is set; caller provides domain-specific search.
// Returns error if zero or multiple sources are provided.
func ResolveIDs(cfg Config, queryFn func(string) ([]string, error)) ([]string, error) {
	sources := 0
	if len(cfg.Args) > 0 {
		sources++
	}
	if cfg.Stdin {
		sources++
	}
	if cfg.Query != "" {
		sources++
	}

	if sources == 0 {
		return nil, fmt.Errorf("provide message IDs as arguments, via --stdin, or via --query")
	}
	if sources > 1 {
		return nil, fmt.Errorf("only one input source allowed: positional args, --stdin, or --query")
	}

	if len(cfg.Args) > 0 {
		return cfg.Args, nil
	}

	if cfg.Stdin {
		return readStdin()
	}

	return queryFn(cfg.Query)
}

func readStdin() ([]string, error) {
	var ids []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			ids = append(ids, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no IDs received from stdin")
	}
	return ids, nil
}
