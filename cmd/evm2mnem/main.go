package main

import (
	"fmt"
	"os"

	"github.com/hugo-dc/ethscripts/common"
)

func showUsage() {
	fmt.Println("evm2mnem - Converts EVM Bytecode to mnemonics")
	fmt.Println("Usage:")
	fmt.Println("\tevm2mnem <bytecode>")
}

func main() {
	if len(os.Args) != 2 {
		showUsage()
		return
	}

	bytecode := os.Args[1]
	mnem := common.Evm2Mnem(bytecode)
	fmt.Println(mnem)
}
