package prober_test

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/infra/prober"
)

func TestGuard_CheckURL_RejectsNonHTTPSchemes(t *testing.T) {
	t.Parallel()

	guard := prober.NewGuard(false)

	for _, target := range []string{
		"file:///etc/passwd",
		"gopher://example.com:70/",
		"ftp://example.com/",
		"redis://example.com:6379",
		"example.com",
	} {
		_, err := guard.CheckURL(target)
		require.ErrorIs(t, err, domain.ErrInvalidTarget, target)
	}
}

func TestGuard_CheckURL_RejectsCredentialsInURL(t *testing.T) {
	t.Parallel()

	_, err := prober.NewGuard(false).CheckURL("https://user:secret@example.com/health")
	require.ErrorIs(t, err, domain.ErrInvalidTarget)
}

func TestGuard_CheckURL_RejectsEmptyHost(t *testing.T) {
	t.Parallel()

	_, err := prober.NewGuard(false).CheckURL("http:///health")
	require.ErrorIs(t, err, domain.ErrInvalidTarget)
}

func TestGuard_CheckURL_RejectsLiteralPrivateAddresses(t *testing.T) {
	t.Parallel()

	guard := prober.NewGuard(false)

	for _, target := range []string{
		"http://127.0.0.1:8080/",
		"http://localhost.localdomain/", // имя не резолвится здесь, проверка на этапе соединения
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5/admin",
		"http://192.168.1.1/",
		"http://172.16.0.1/",
		"http://[::1]:5432/",
		"http://[fd00::1]/",
		"http://100.64.0.1/",
		"http://0.0.0.0/",
	} {
		_, err := guard.CheckURL(target)

		if target == "http://localhost.localdomain/" {
			continue
		}

		require.ErrorIs(t, err, domain.ErrTargetNotAllowed, target)
	}
}

func TestGuard_CheckURL_AllowsPublicAddresses(t *testing.T) {
	t.Parallel()

	guard := prober.NewGuard(false)

	for _, target := range []string{
		"https://example.com/health",
		"http://93.184.216.34/",
		"https://api.github.com:443/",
		"https://example.com/path?query=1",
	} {
		parsed, err := guard.CheckURL(target)
		require.NoError(t, err, target)
		require.NotNil(t, parsed)
	}
}

func TestGuard_AllowPrivatePermitsLoopback(t *testing.T) {
	t.Parallel()

	parsed, err := prober.NewGuard(true).CheckURL("http://127.0.0.1:8080/healthz")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:8080", parsed.Host)
}

func TestGuard_CheckDialAddress(t *testing.T) {
	t.Parallel()

	guard := prober.NewGuard(false)

	require.ErrorIs(t, guard.CheckDialAddress("127.0.0.1:80"), domain.ErrTargetNotAllowed)
	require.ErrorIs(t, guard.CheckDialAddress("169.254.169.254:80"), domain.ErrTargetNotAllowed)
	require.ErrorIs(t, guard.CheckDialAddress("[::1]:443"), domain.ErrTargetNotAllowed)
	require.NoError(t, guard.CheckDialAddress("93.184.216.34:443"))

	require.ErrorIs(t, guard.CheckDialAddress("not-an-address"), domain.ErrTargetNotAllowed)
}

func TestGuard_CheckAddr_UnmapsIPv4MappedAddresses(t *testing.T) {
	t.Parallel()

	addr := netip.MustParseAddr("::ffff:127.0.0.1")

	require.ErrorIs(t, prober.NewGuard(false).CheckAddr(addr), domain.ErrTargetNotAllowed)
}
