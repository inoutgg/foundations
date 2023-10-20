package github

import (
	"golang.org/x/oauth2"
)

type UserInfo interface {
  ID() string
}

type Provider struct {
  UserInfo() (UserInfo, error)
}

func New() Provider {
  return &github{}
}

type github struct {}

func (g *github) UserInfo() (UserInfo, error) {
  return nil, nil
}
