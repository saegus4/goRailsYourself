package crypto

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strings"
)

// MessageVerifier makes it easy to generate and verify messages which are
// signed to prevent tampering.
//
// This is useful for cases like remember-me tokens and auto-unsubscribe links
// where the session store isn't suitable or available.
type MessageVerifier struct {
	// Secret of 32-bytes if using the default hashing.
	Secret []byte
	// Hasher defaults to sha1 if not set.
	Hasher func() hash.Hash
	// Serializer defines the way the data is serializer/deserialized.
	Serializer MsgSerializer
}

// Checks that the struct is properly set and ready for use.
func (crypt *MessageVerifier) IsValid() (bool, error) {
	err := crypt.checkInit()
	if err != nil {
		return false, err
	}
	return true, nil
}

// Verify() takes a base64 encoded message string joined to a digest by a double dash "--"
// and returns an error if anything wrong happen.
// If the verification worked, the target interface object passed is populated.
func (crypt *MessageVerifier) Verify(msg string, target interface{}) error {
	// TODO: check that the target is a pointer.
	err := crypt.checkInit()
	if err != nil {
		return err
	}

	invalid := func(msg string) error {
		return errors.New("Invalid signature - " + msg)
	}
	if msg == "" {
		return invalid("empty message")
	}

	dataDigest := strings.Split(msg, "--")
	if len(dataDigest) != 2 {
		return invalid("bad data --")
	}

	data, digest := dataDigest[0], dataDigest[1]
	if crypt.secureCompare(digest, crypt.DigestFor(data)) == false {
		return invalid("bad data (compare)")
	}
	decodedData, _ := base64.StdEncoding.Strict().DecodeString(data)
  decodedString := "\"" + string(decodedData) + "\""
	err = crypt.Serializer.Unserialize(string(decodedString), target)

  if err != nil {
    err = crypt.Serializer.Unserialize(string(decodedData), target)
    if err != nil {
      return fmt.Errorf("failed to unserialize both quoted and raw data: %w", err)
    }
  }

	return err
}

// Generate() Converts an interface into a string containing the serialized data
// and a digest.
// The string can be passed around and tampering can be checked using the digest.
// See Verify() to extract the data out of the signed string.
func (crypt *MessageVerifier) Generate(value interface{}) (string, error) {
	err := crypt.checkInit()
	if err != nil {
		return "", err
	}

	data, err := crypt.Serializer.Serialize(value)
	if err != nil {
		return "", err
	}
	str := base64.StdEncoding.EncodeToString([]byte(data))
	digest := crypt.DigestFor(str)
	return fmt.Sprintf("%s--%s", str, digest), nil
}

// DigestFor returns the digest form of a string after hashing it via
// the verifier's digest and secret.
func (crypt *MessageVerifier) DigestFor(data string) string {
	if crypt.Secret == nil {
		return "Y U SET NO SECRET???!"
	}

	mac := hmac.New(crypt.Hasher, crypt.Secret)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// constant-time comparison algorithm to prevent timing attacks
func (crypt *MessageVerifier) secureCompare(strA, strB string) bool {
	a := []byte(strA)
	b := []byte(strB)

	if len(a) != len(b) {
		return false
	}
	res := 0
	for i := 0; i < len(a); i++ {
		res |= int(b[i]) ^ int(a[i])
	}
	return res == 0
}

func (crypt *MessageVerifier) checkInit() error {
	if crypt == nil {
		return errors.New("MessageVerifier not set")
	}
	if crypt.Serializer == nil {
		return errors.New("Serializer not set")
	}

	if crypt.Hasher == nil {
		// set a default hasher
		crypt.Hasher = sha1.New
	}

	if crypt.Secret == nil {
		return errors.New("Secret not set")
	}

	return nil
}
