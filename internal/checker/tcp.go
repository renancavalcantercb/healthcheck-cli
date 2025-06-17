package checker

import (
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

type TCPChecker struct{}

func NewTCPChecker(timeout time.Duration) *TCPChecker {
	return &TCPChecker{}
}

func (t *TCPChecker) Name() string {
	return "TCP"
}

func (t *TCPChecker) Check(check types.CheckConfig) types.Result {
	return types.Result{
		Name:   check.Name,
		URL:    check.URL,
		Status: types.StatusUp,
	}
}
