package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrustedProxyList_IsTrustedProxy_WithPort(t *testing.T) {
	trustedProxy, err := NewTrustedProxyList([]string{"10.0.0.0/8", "172.16.0.0/12"})
	assert.NoError(t, err)

	assert.True(t, trustedProxy.IsTrustedProxy("10.0.1.23:54321"))
	assert.True(t, trustedProxy.IsTrustedProxy("172.16.1.1:8080"))
	assert.False(t, trustedProxy.IsTrustedProxy("192.168.1.1:8080"))
}

func TestTrustedProxyList_IsTrustedProxy_WithoutPort(t *testing.T) {
	trustedProxy, err := NewTrustedProxyList([]string{"10.0.0.0/8"})
	assert.NoError(t, err)

	assert.True(t, trustedProxy.IsTrustedProxy("10.0.1.23"))
	assert.False(t, trustedProxy.IsTrustedProxy("192.168.1.1"))
}

func TestTrustedProxyList_IsTrustedProxy_EmptyList(t *testing.T) {
	trustedProxy, err := NewTrustedProxyList([]string{})
	assert.NoError(t, err)

	assert.False(t, trustedProxy.IsTrustedProxy("10.0.1.23:54321"))
	assert.False(t, trustedProxy.IsTrustedProxy("192.168.1.1:8080"))
}

func TestTrustedProxyList_IsTrustedProxy_InvalidAddress(t *testing.T) {
	trustedProxy, err := NewTrustedProxyList([]string{"10.0.0.0/8"})
	assert.NoError(t, err)

	assert.False(t, trustedProxy.IsTrustedProxy("invalid-address"))
	assert.False(t, trustedProxy.IsTrustedProxy(""))
}

func TestTrustedProxyList_NewTrustedProxyList_InvalidCIDR(t *testing.T) {
	_, err := NewTrustedProxyList([]string{"invalid-cidr"})
	assert.Error(t, err)
}

func TestTrustedProxyList_NewTrustedProxyList_EmptyCIDR(t *testing.T) {
	trustedProxy, err := NewTrustedProxyList([]string{"", "10.0.0.0/8"})
	assert.NoError(t, err)
	assert.True(t, trustedProxy.IsTrustedProxy("10.0.1.23:54321"))
}
