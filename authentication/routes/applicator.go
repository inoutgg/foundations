package routes

import (
	"github.com/atcirclesquare/common/http/errorhandler"
	"github.com/atcirclesquare/common/http/routerutil"
)

type Config struct {
	Prod         bool
	ErrorHandler errorhandler.ErrorHandler
}

type Applicator interface {
	Routes(*Config) routerutil.Applicator
}
