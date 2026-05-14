package main

import (
	"crypto/hkdf"
	"crypto/mlkem"
	"crypto/sha256"
	"fmt"
	"log"
)

func main() {
	// Generate KEM
	decapsulationKey, err := mlkem.GenerateKey768()
	if err != nil {
		log.Fatal(err)
	}

	encapsulationKey := decapsulationKey.EncapsulationKey().Bytes()
	ciphertext := B(encapsulationKey)
	sharedKey, err := decapsulationKey.Decapsulate(ciphertext)
	_ = err

	fmt.Println(sharedKey)

	sha := sha256.New
	sharedKey, err = hkdf.Key(sha, sharedKey, nil, "", len(sharedKey))
	fmt.Println(sharedKey)

	sharedKey, err = hkdf.Key(sha, sharedKey, nil, "", len(sharedKey))
	fmt.Println(sharedKey)
}

func B(encapsulationKey []byte) (ciphertext []byte) {
	ek, err := mlkem.NewEncapsulationKey768(encapsulationKey)
	if err != nil {
		log.Fatal(err)
	}
	sharedKey, ciphertext := ek.Encapsulate()
	fmt.Println(sharedKey)

	sha := sha256.New
	sharedKey, err = hkdf.Key(sha, sharedKey, nil, "", len(sharedKey))
	fmt.Println(sharedKey)

	sharedKey, err = hkdf.Key(sha, sharedKey, nil, "", len(sharedKey))
	fmt.Println(sharedKey)
	return ciphertext
}
