package auth

import "golang.org/x/crypto/bcrypt"

// hashPassword hashes a plaintext password with bcrypt. Callers must ensure
// password is <=72 bytes first (bcrypt.GenerateFromPassword hard-errors above
// that) — see validateUsernamePassword.
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// verifyPassword reports whether password matches the bcrypt hash.
func verifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
