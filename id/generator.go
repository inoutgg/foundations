package id

import (
	"github.com/segmentio/ksuid"
)

var _ Generator = (*generator)(nil)
var _ ID = (*id)(nil)

// ID represents a unique identifier.
type ID interface {
	// String returns the string representation of the ID.
	String() string

	// Next returns a new ID that is guaranteed to be greater than the current ID.
	// It returns false if the current ID is already the maximum ID.
	Next() (bool, ID)
}

type Generator interface {
	// Generate returns a new unique ID.
	Generate() ID
}

// New returns a new ID generator based on top of KSUID.
func New() Generator {
	return generator{}
}

type generator struct{}

func (g generator) Generate() ID {
	uid := ksuid.New()

	return &id{
		uid: uid,
		seq: nil,
	}
}

type id struct {
	uid ksuid.KSUID
	seq *ksuid.Sequence
}

func (i *id) String() string { return i.uid.String() }

func (i *id) Next() (bool, ID) {
	// Lazy init the sequence
	if i.seq == nil {
		i.seq = &ksuid.Sequence{Seed: i.uid}
	}

	uid, err := i.seq.Next()
	if err != nil {
		return false, nil
	}

	return true, &id{uid: uid, seq: i.seq}
}
