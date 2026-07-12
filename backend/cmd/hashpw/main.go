// hashpw generates the bcrypt hash for APP_PASSWORD_HASH:
//
//	go run ./cmd/hashpw 'your-password'
package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: hashpw <password>")
		os.Exit(2)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash:", err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}
