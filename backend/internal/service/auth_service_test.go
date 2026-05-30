package service_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/mocks"
	"github.com/jose/ratiodash/internal/service"
)

// validHash returns a bcrypt hash of password for use in test fixtures.
func validHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func TestAuthService_IsSetup(t *testing.T) {
	t.Run("returns true when credential exists", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{Username: "admin"}, nil)

		ok, err := service.NewAuthService(repo).IsSetup()

		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("returns false when no credential exists", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, nil)

		ok, err := service.NewAuthService(repo).IsSetup()

		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, errors.New("db error"))

		_, err := service.NewAuthService(repo).IsSetup()

		assert.Error(t, err)
	})
}

func TestAuthService_Setup(t *testing.T) {
	t.Run("creates credential when none exist", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, nil)
		repo.EXPECT().Create("admin", mock_anyString, mock_anyString).Return(&domain.AppCredential{}, nil)

		err := service.NewAuthService(repo).Setup("admin", "password123")

		require.NoError(t, err)
	})

	t.Run("returns error when already configured", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{Username: "admin"}, nil)

		err := service.NewAuthService(repo).Setup("admin", "password")

		assert.ErrorContains(t, err, "already configured")
	})

	t.Run("propagates Find error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, errors.New("db error"))

		err := service.NewAuthService(repo).Setup("admin", "password")

		assert.Error(t, err)
	})

	t.Run("propagates Create error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, nil)
		repo.EXPECT().Create("admin", mock_anyString, mock_anyString).Return(nil, errors.New("db error"))

		err := service.NewAuthService(repo).Setup("admin", "password123")

		assert.Error(t, err)
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("returns signed JWT on valid credentials", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{
			Username:     "admin",
			PasswordHash: validHash(t, "secret"),
			JWTSecret:    "jwt-signing-secret",
		}, nil)

		token, err := service.NewAuthService(repo).Login("admin", "secret")

		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("returns error when not configured", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, nil)

		_, err := service.NewAuthService(repo).Login("admin", "secret")

		assert.ErrorContains(t, err, "not configured")
	})

	t.Run("returns error on wrong username", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{
			Username:     "admin",
			PasswordHash: validHash(t, "secret"),
			JWTSecret:    "jwt-signing-secret",
		}, nil)

		_, err := service.NewAuthService(repo).Login("wrong", "secret")

		assert.ErrorContains(t, err, "invalid credentials")
	})

	t.Run("returns error on wrong password", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{
			Username:     "admin",
			PasswordHash: validHash(t, "secret"),
			JWTSecret:    "jwt-signing-secret",
		}, nil)

		_, err := service.NewAuthService(repo).Login("admin", "wrong")

		assert.ErrorContains(t, err, "invalid credentials")
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, errors.New("db error"))

		_, err := service.NewAuthService(repo).Login("admin", "secret")

		assert.Error(t, err)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	const secret = "jwt-signing-secret"
	const username = "admin"

	makeToken := func(t *testing.T) string {
		t.Helper()
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": username})
		signed, err := tok.SignedString([]byte(secret))
		require.NoError(t, err)
		return signed
	}

	t.Run("returns subject from valid token", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{
			Username:  username,
			JWTSecret: secret,
		}, nil)

		sub, err := service.NewAuthService(repo).ValidateToken(makeToken(t))

		require.NoError(t, err)
		assert.Equal(t, username, sub)
	})

	t.Run("returns error when not configured", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, nil)

		_, err := service.NewAuthService(repo).ValidateToken("any")

		assert.ErrorContains(t, err, "not configured")
	})

	t.Run("returns error for invalid token", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{
			Username:  username,
			JWTSecret: secret,
		}, nil)

		_, err := service.NewAuthService(repo).ValidateToken("not.a.valid.token")

		assert.Error(t, err)
	})

	t.Run("returns error for non-HMAC signed token", func(t *testing.T) {
		// Generate a throwaway RSA key so the token has alg=RS256,
		// which should trigger the unexpected signing method branch.
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": username})
		rsaToken, err := tok.SignedString(rsaKey)
		require.NoError(t, err)

		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{
			Username:  username,
			JWTSecret: secret,
		}, nil)

		_, err = service.NewAuthService(repo).ValidateToken(rsaToken)

		assert.Error(t, err)
	})

	t.Run("returns error when sub claim is not a string", func(t *testing.T) {
		// Craft a token where sub is a number rather than a string.
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": 42})
		numericSubToken, err := tok.SignedString([]byte(secret))
		require.NoError(t, err)

		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{
			Username:  username,
			JWTSecret: secret,
		}, nil)

		_, err = service.NewAuthService(repo).ValidateToken(numericSubToken)

		assert.Error(t, err)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, errors.New("db error"))

		_, err := service.NewAuthService(repo).ValidateToken(makeToken(t))

		assert.Error(t, err)
	})
}

func TestAuthService_UpdateCredentials(t *testing.T) {
	t.Run("updates with correct current password", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		cred := &domain.AppCredential{
			ID:           1,
			Username:     "admin",
			PasswordHash: validHash(t, "oldpass"),
		}
		repo.EXPECT().Find().Return(cred, nil)
		repo.EXPECT().Update(uint(1), "newadmin", mock_anyString).Return(nil)

		err := service.NewAuthService(repo).UpdateCredentials("oldpass", "newadmin", "newpass")

		require.NoError(t, err)
	})

	t.Run("returns error when not configured", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, nil)

		err := service.NewAuthService(repo).UpdateCredentials("old", "new", "newpass")

		assert.ErrorContains(t, err, "not configured")
	})

	t.Run("returns error on wrong current password", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{
			ID:           1,
			PasswordHash: validHash(t, "correct"),
		}, nil)

		err := service.NewAuthService(repo).UpdateCredentials("wrong", "newadmin", "newpass")

		assert.ErrorContains(t, err, "invalid current password")
	})

	t.Run("propagates Find error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, errors.New("db error"))

		err := service.NewAuthService(repo).UpdateCredentials("old", "new", "newpass")

		assert.Error(t, err)
	})

	t.Run("propagates Update error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{
			ID:           1,
			PasswordHash: validHash(t, "oldpass"),
		}, nil)
		repo.EXPECT().Update(uint(1), "newadmin", mock_anyString).Return(errors.New("db error"))

		err := service.NewAuthService(repo).UpdateCredentials("oldpass", "newadmin", "newpass")

		assert.Error(t, err)
	})
}

func TestAuthService_GetLanguage(t *testing.T) {
	t.Run("returns default en when not configured", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, nil)

		language, err := service.NewAuthService(repo).GetLanguage()

		require.NoError(t, err)
		assert.Equal(t, "en", language)
	})

	t.Run("returns normalized stored language", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{Language: "FR"}, nil)

		language, err := service.NewAuthService(repo).GetLanguage()

		require.NoError(t, err)
		assert.Equal(t, "fr", language)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, errors.New("db error"))

		_, err := service.NewAuthService(repo).GetLanguage()

		assert.Error(t, err)
	})
}

func TestAuthService_UpdateLanguage(t *testing.T) {
	t.Run("updates normalized language", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{ID: 1}, nil)
		repo.EXPECT().UpdateLanguage(uint(1), "fr").Return(nil)

		err := service.NewAuthService(repo).UpdateLanguage(" FR ")

		require.NoError(t, err)
	})

	t.Run("defaults unknown language to en", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{ID: 1}, nil)
		repo.EXPECT().UpdateLanguage(uint(1), "en").Return(nil)

		err := service.NewAuthService(repo).UpdateLanguage("es")

		require.NoError(t, err)
	})

	t.Run("returns error when not configured", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, nil)

		err := service.NewAuthService(repo).UpdateLanguage("fr")

		assert.ErrorContains(t, err, "not configured")
	})

	t.Run("propagates find error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(nil, errors.New("db error"))

		err := service.NewAuthService(repo).UpdateLanguage("fr")

		assert.Error(t, err)
	})

	t.Run("propagates update error", func(t *testing.T) {
		repo := mocks.NewMockAuthRepository(t)
		repo.EXPECT().Find().Return(&domain.AppCredential{ID: 1}, nil)
		repo.EXPECT().UpdateLanguage(uint(1), "fr").Return(errors.New("db error"))

		err := service.NewAuthService(repo).UpdateLanguage("fr")

		assert.Error(t, err)
	})
}

// mock_anyString matches any string argument in EXPECT() calls where the
// exact value cannot be known in advance (e.g. bcrypt hashes, JWT secrets).
var mock_anyString = mock.AnythingOfType("string")
