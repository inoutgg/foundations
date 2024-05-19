package password

var (
	// DefaultPasswordHasher is the default password hashing algorithm used across.
	DefaultPasswordHasher = NewBcryptPasswordHasher(BcryptDefaultCost)
)

// PasswordHasher is a hashing algorithm to hash password securely.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hashedPassword string, password string) (bool, error)
}
