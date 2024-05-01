package password

// PasswordHasher is a hashing algorithm to hash password securely.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hashedPassword string, password string) (bool, error)
}
