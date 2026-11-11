### internal/admin/keys.go
package admin

import (
	"fmt"
	"strings"

	gossh "golang.org/x/crypto/ssh"
)

func ParseKey(text string) (fingerprint, authorized string, err error) {
	pub, _, _, _, perr := gossh.ParseAuthorizedKey([]byte(strings.TrimSpace(text)))
	if perr != nil {
		return "", "", fmt.Errorf("that is not a valid SSH public key: %w", perr)
	}
	return gossh.FingerprintSHA256(pub), strings.TrimSpace(string(gossh.MarshalAuthorizedKey(pub))), nil
}
