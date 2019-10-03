package main

import (
	b64 "encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	username := flag.String("username", "", "Username for the account.")
	password := flag.String("password", "", "Password for the account.")
	hash := flag.String("hash", "", "Hash of the password for the account.")
	op := flag.String("op", "create", "Operation type. create or verify")
	role := flag.Int("role", 1, "The account role. 1 is readonly and 5 is poweruser.")
	flag.Parse()

	if *op == "verify" {
		if *hash == "" || *password == "" {
			log.Fatal("Hash and/or password can't be empty")
		}

		err := bcrypt.CompareHashAndPassword([]byte(*hash), []byte(*password))
		if err != nil {
			log.Println(err)
			log.Println(*hash)
			fmt.Println(false)
			os.Exit(0)
		}

		fmt.Println(true)
		os.Exit(0)
	}

	if *username == "" || *password == "" {
		log.Fatal("Username and/or password need to be provided.")
	}

	hashNew, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	hashEnc := b64.StdEncoding.EncodeToString([]byte(hashNew))
	usernameEnc := b64.StdEncoding.EncodeToString([]byte(*username))

	fmt.Printf("%s,%d,%s\n", usernameEnc, *role, hashEnc)
}
