package auth

import "errors"

var (
	// ErrUsernameTaken is returned by Register when the username already exists.
	ErrUsernameTaken = errors.New("username already taken")
	// ErrInvalidCredentials is returned by Login for both an unknown username
	// and a wrong password, so a failed login can't be used to enumerate
	// which usernames exist.
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrInvalidUsername / ErrInvalidPassword are returned by Register for
	// basic input validation failures.
	ErrInvalidUsername = errors.New("username must be 3-32 characters")
	ErrInvalidPassword = errors.New("password must be 8-72 characters")
	// ErrUserNotFound is returned by GetUser when the ID doesn't exist.
	ErrUserNotFound = errors.New("user not found")
)
