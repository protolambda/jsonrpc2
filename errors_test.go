package jsonrpc

import (
	"strings"
	"testing"
)

func TestIsServerError(t *testing.T) {
	for _, c := range []ErrorConst{-32000, -32001, -32050, -32098, -32099, InvalidInput, JSONRPCVersionNotSupported} {
		if !c.IsServerError() {
			t.Errorf("expected server error: %d", c)
		}
	}
	for _, c := range []ErrorConst{-31999, -32100, 0, ParseErr, InternalError, UserRejectedRequest} {
		if c.IsServerError() {
			t.Errorf("expected no server error: %d", c)
		}
	}
}

func TestErrorConstMessage(t *testing.T) {
	known := []ErrorConst{
		ParseErr, InvalidRequest, MethodNotFound, InvalidParams, InternalError,
		InvalidInput, ResourceNotFound, ResourceUnavailable, TransactionRejected, MethodNotSupported, LimitExceeded,
		JSONRPCVersionNotSupported,
		UserRejectedRequest, Unauthorized, UnsupportedMethod, Disconnected, ChainDisconnected, UnrecognizedChainID,
		UnsupportedNonOptionalCapability, UnsupportedChainID, DuplicateBundleID, UnknownBundleID, BundleTooLarge,
		AtomicReadyWalletRejectedUpgrade, AtomicityNotSupported,
	}
	for _, c := range known {
		if strings.HasPrefix(c.Message(), "Non-standard") {
			t.Errorf("expected a standard message for %d", c)
		}
	}
	if got, want := ErrorConst(1234).Message(), "Non-standard error-code 1234"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
