package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run tools/hashpass.go <password>")
		fmt.Println("Example: go run tools/hashpass.go mysecretpassword")
		os.Exit(1)
	}

	password := os.Args[1]

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Password Hash:")
	fmt.Println(string(hash))
	fmt.Println()
	fmt.Println("Add this to your config.yaml:")
	fmt.Printf("webui:\n  authentication:\n    enabled: true\n    username: admin\n    password_hash: \"%s\"\n    session_timeout: 60\n    secret_key: \"your-random-secret-key-here\"\n", string(hash))
}
