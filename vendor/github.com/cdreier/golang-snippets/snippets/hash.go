package snippets

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
)

// MD5Hash just hashes given string
func MD5Hash(in string) string {
	hasher := md5.New()
	hasher.Write([]byte(in))
	return hex.EncodeToString(hasher.Sum(nil))
}

// SHA256Hash just hashes given string
func SHA256Hash(in string) string {
	hasher := sha256.New()
	hasher.Write([]byte(in))
	return hex.EncodeToString(hasher.Sum(nil))
}
