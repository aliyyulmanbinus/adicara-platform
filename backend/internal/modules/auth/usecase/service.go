package usecase

import (
	"context"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/domain"
)

const (
	minUsernameLen = 3
	maxUsernameLen = 30
	maxEmailLen    = 254
	minPasswordLen = 8
	// bcrypt rejects anything longer than 72 bytes.
	maxPasswordBytes = 72
	bcryptCost       = 12

	// A refresh token spent by rotation this recently is far more likely a
	// client race (two tabs, a retried request) than theft, so it is only
	// rejected. Replayed after this, the token is assumed stolen and every
	// session of the user is revoked. The frontend coalesces refreshes for 10s.
	refreshReuseGrace = 10 * time.Second
)

var (
	usernamePattern   = regexp.MustCompile(`^[a-z0-9._]+$`)
	passwordHasLetter = regexp.MustCompile(`\p{L}`)
	passwordHasDigit  = regexp.MustCompile(`\d`)
)

// timingDummyHash is a real bcrypt hash at the production cost. Login compares
// against it when the identifier is unknown so that case costs as much as a
// wrong password and response time does not reveal which accounts exist.
var timingDummyHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("timing-equalizer-not-a-real-password"), bcryptCost)
	if err != nil {
		panic("bcrypt: " + err.Error())
	}

	return hash
})

type Service struct {
	repo       domain.Repository
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func New(repo domain.Repository, secret string, accessTTL, refreshTTL time.Duration) *Service {
	timingDummyHash() // pay for it at start, not on the first unknown login

	return &Service{repo: repo, secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (s *Service) Register(ctx context.Context, email, password, username string) (domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.ToLower(strings.TrimSpace(username))
	if email == "" || username == "" {
		return domain.User{}, apierr.BadRequest("invalid_input", "email or username are required")
	}
	if !validEmail(email) {
		return domain.User{}, apierr.BadRequest("invalid_email", "email address is not valid")
	}
	if len(username) < minUsernameLen || len(username) > maxUsernameLen || !usernamePattern.MatchString(username) {
		return domain.User{}, apierr.BadRequest("invalid_username", "username must be 3 to 30 characters and contain only lowercase letters, digits, dots, or underscores")
	}
	if err := validatePassword(password); err != nil {
		return domain.User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return domain.User{}, err
	}

	user := domain.User{ID: uuid.New(), Email: email, Username: username, RoleID: domain.RoleCustomer, Status: domain.StatusActive}
	if err := s.repo.CreateUser(ctx, user.ID, user.Email, user.Username, string(hash)); err != nil {
		switch {
		case errors.Is(err, domain.ErrUsernameTaken):
			return domain.User{}, apierr.Conflict("username_taken", "username already registered")
		case errors.Is(err, domain.ErrEmailTaken):
			return domain.User{}, apierr.Conflict("email_taken", "email already registered")
		default:
			return domain.User{}, err
		}
	}

	return user, nil
}

func validEmail(email string) bool {
	if len(email) > maxEmailLen {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	at := strings.LastIndex(email, "@")

	return strings.Contains(email[at+1:], ".")
}

func validatePassword(password string) error {
	if len(password) < minPasswordLen || !passwordHasLetter.MatchString(password) || !passwordHasDigit.MatchString(password) {
		return apierr.BadRequest("weak_password", "password must be at least 8 characters and contain both letters and digits")
	}
	if len(password) > maxPasswordBytes {
		return apierr.BadRequest("password_too_long", "password must be at most 72 bytes")
	}

	return nil
}

func (s *Service) Login(ctx context.Context, identifier, password string) (domain.User, domain.TokenPair, error) {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	user, hash, err := s.repo.UserByIdentifier(ctx, identifier)
	if errors.Is(err, domain.ErrUserNotFound) {
		_ = bcrypt.CompareHashAndPassword(timingDummyHash(), []byte(password))

		return domain.User{}, domain.TokenPair{}, apierr.Unauthorized("invalid credentials")
	}
	if err != nil {
		return domain.User{}, domain.TokenPair{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return domain.User{}, domain.TokenPair{}, apierr.Unauthorized("invalid credentials")
	}
	if user.Status != domain.StatusActive {
		return domain.User{}, domain.TokenPair{}, apierr.New(403, "account_inactive", "account is inactive")
	}
	tokens, err := s.issue(ctx, user.ID)

	return user, tokens, err
}

// Refresh rotates the pair: the presented session is spent, so a refresh token
// that leaks and is replayed after the owner refreshed is rejected. A replay
// that is not just a client race also revokes every session of the user, since
// either the owner or the thief now holds a dead token and one of them has the
// live one.
func (s *Service) Refresh(ctx context.Context, token string) (domain.TokenPair, error) {
	userID, sessionID, err := s.parseRefresh(token)
	if err != nil {
		return domain.TokenPair{}, err
	}

	if err := s.repo.ConsumeRefreshSession(ctx, sessionID, userID); err != nil {
		if errors.Is(err, domain.ErrSessionInactive) {
			if _, err := s.repo.RevokeUserSessionsOnReuse(ctx, sessionID, userID, refreshReuseGrace); err != nil {
				return domain.TokenPair{}, err
			}

			return domain.TokenPair{}, apierr.Unauthorized("refresh token is no longer valid")
		}
		return domain.TokenPair{}, err
	}

	// 401 rather than 403: a client that treats 401 as "log in again" ends up
	// at the login form, which then explains that the account is inactive.
	if err := s.requireActive(ctx, userID); err != nil {
		return domain.TokenPair{}, err
	}

	return s.issue(ctx, userID)
}

// requireActive fails with 401 when the account is gone or deactivated.
func (s *Service) requireActive(ctx context.Context, userID uuid.UUID) error {
	state, err := s.repo.UserAuthState(ctx, userID)
	if errors.Is(err, domain.ErrUserNotFound) || (err == nil && state.Status != domain.StatusActive) {
		return apierr.Unauthorized("account is not active")
	}

	return err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	userID, sessionID, err := s.parseRefresh(token)
	if err != nil {
		return err
	}

	return s.repo.RevokeRefreshSession(ctx, sessionID, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, name string) (domain.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.User{}, apierr.BadRequest("invalid_input", "name is required")
	}

	user, err := s.repo.UpdateUserName(ctx, id, name)
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, apierr.NotFound("user")
	}

	return user, err
}

// ChangePassword revokes every session: a password change is also the way to
// kick out a device whose refresh token was lost.
func (s *Service) ChangePassword(ctx context.Context, id uuid.UUID, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}

	hash, err := s.repo.PasswordHash(ctx, id)
	if errors.Is(err, domain.ErrUserNotFound) {
		return apierr.NotFound("user")
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		return apierr.Unauthorized("current password is incorrect")
	}

	fresh, err := bcrypt.GenerateFromPassword([]byte(next), bcryptCost)
	if err != nil {
		return err
	}

	return s.repo.ChangePassword(ctx, id, string(fresh), time.Now())
}

// Authenticate turns a bearer access token into a user id. Besides the
// signature it checks the database, so deactivating an account or changing its
// password takes effect on the next request instead of when the token expires.
// A failure is always 401 (see requireActive for why not 403).
func (s *Service) Authenticate(ctx context.Context, token string) (uuid.UUID, error) {
	claims, err := s.parse(token)
	if err != nil || claims["typ"] != "access" {
		return uuid.Nil, apierr.Unauthorized("invalid access token")
	}

	sub, _ := claims["sub"].(string)
	id, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, apierr.Unauthorized("invalid access token")
	}

	state, err := s.repo.UserAuthState(ctx, id)
	if errors.Is(err, domain.ErrUserNotFound) {
		return uuid.Nil, apierr.Unauthorized("account is not active")
	}
	if err != nil {
		return uuid.Nil, err
	}
	if state.Status != domain.StatusActive {
		return uuid.Nil, apierr.Unauthorized("account is not active")
	}
	if state.PasswordChangedAt != nil {
		// iat has one-second resolution, so a token issued in the same second as
		// the change is accepted rather than risking a just-logged-in user.
		if iat, ok := claims["iat"].(float64); !ok || int64(iat) < state.PasswordChangedAt.Unix() {
			return uuid.Nil, apierr.Unauthorized("access token predates the last password change")
		}
	}

	return id, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.User, error) {
	user, err := s.repo.UserByID(ctx, id)
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, apierr.NotFound("user")
	}

	return user, err
}

func (s *Service) issue(ctx context.Context, userID uuid.UUID) (domain.TokenPair, error) {
	access, err := s.sign(userID, "access", s.accessTTL, uuid.Nil)
	if err != nil {
		return domain.TokenPair{}, err
	}

	sessionID := uuid.New()
	refresh, err := s.sign(userID, "refresh", s.refreshTTL, sessionID)
	if err != nil {
		return domain.TokenPair{}, err
	}
	if err := s.repo.CreateRefreshSession(ctx, sessionID, userID, time.Now().Add(s.refreshTTL)); err != nil {
		return domain.TokenPair{}, err
	}

	return domain.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(s.accessTTL.Seconds()),
	}, nil
}

func (s *Service) sign(userID uuid.UUID, typ string, ttl time.Duration, sessionID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"typ": typ,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}
	if sessionID != uuid.Nil {
		claims["jti"] = sessionID.String()
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *Service) parseRefresh(token string) (userID, sessionID uuid.UUID, err error) {
	claims, err := s.parse(token)
	if err != nil || claims["typ"] != "refresh" {
		return uuid.Nil, uuid.Nil, apierr.Unauthorized("invalid refresh token")
	}

	sub, _ := claims["sub"].(string)
	userID, err = uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, uuid.Nil, apierr.Unauthorized("invalid refresh token")
	}

	jti, _ := claims["jti"].(string)
	sessionID, err = uuid.Parse(jti)
	if err != nil {
		return uuid.Nil, uuid.Nil, apierr.Unauthorized("invalid refresh token")
	}

	return userID, sessionID, nil
}

// PurgeExpiredSessions drops rows that can no longer authorise anything.
func (s *Service) PurgeExpiredSessions(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpiredSessions(ctx)
}

func (s *Service) parse(token string) (jwt.MapClaims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, apierr.Unauthorized("invalid token")
		}
		return s.secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, apierr.Unauthorized("invalid token")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, apierr.Unauthorized("invalid token")
	}

	return claims, nil
}
