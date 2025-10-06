// SPDX-FileCopyrightText: 2025 GSI Helmholtzzentrum für Schwerionenforschung GmbH
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Extract tool name from argv[0]
	toolName := filepath.Base(os.Args[0])
	args := os.Args[1:]

	// Print identifiable output so tests can verify this mock was executed
	fmt.Printf("MOCK-%s-EXECUTED\n", strings.ToUpper(toolName))

	if len(args) > 0 {
		fmt.Printf("ARGS: %s\n", strings.Join(args, " "))
	}
}
