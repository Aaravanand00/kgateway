package stringutils

import (
	"crypto/sha256"
	"fmt"
	slices0 "slices"

	slices "golang.org/x/exp/slices"
)

// Only deletes the first instance of value!
// Takes a slice and a value and if that value is found, uses Delete from the exp.slices package to remove it.
// Otherwise returns the original slice.
func DeleteOneByValue(slice []string, value string) []string {
	index := slices.Index(slice, value)
	if index == -1 {
		return slice
	}
	return slices.Delete(slice, index, index+1)
}

// AppendIfMissing returns a slice, with the provided value included
// If the value already exists in the slice, it will not be duplicated
func AppendIfMissing(slice []string, value string) []string {
	if slices0.Contains(slice, value) {
		return slice
	}
	return append(slice, value)
}

// TruncateMaxLength returns a string truncated to the specified maximum length.
// If the string is already shorter than or equal to maxLen, it returns the original string.
// Otherwise, it returns the string truncated to maxLen characters.
func TruncateMaxLength(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// HashGatewayName generates a unique name for a Helm release from a Gateway's name, namespace, and UID.
func HashGatewayName(name, namespace, uid string) string {
	h := sha256.New()
	h.Write([]byte(name + "/" + namespace + ":" + uid))
	return fmt.Sprintf("kgw-%x", h.Sum(nil))[:12]
}

// SafeTruncateAndHash truncates a string and appends a hash of the original string if it exceeds maxLen.
func SafeTruncateAndHash(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	h := sha256.New()
	h.Write([]byte(s))
	hashStr := fmt.Sprintf("%x", h.Sum(nil))[:8]

	// Ensure the total length is maxLen by truncating the name further to accommodate the hash and a separator
	return s[:maxLen-len(hashStr)-1] + "-" + hashStr
}
