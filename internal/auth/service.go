package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"codeberg.org/urutau-ltd/aile/v2/x/htmx"
	accountsetting "codeberg.org/urutau-ltd/gavia/internal/models/account_setting"
	"codeberg.org/urutau-ltd/gavia/internal/models/session"
	"codeberg.org/urutau-ltd/gavia/internal/security"
)

const (
	SessionCookieName = "gavia_session"
	SetupPath         = "/account-settings/edit"
	LoginPath         = "/login"
	DashboardPath     = "/dashboard"
	sessionTTL        = 14 * 24 * time.Hour
)

type viewerContextKey struct{}

type Viewer struct {
	HasAccount         bool
	IsAuthenticated    bool
	IsAPIAuthenticated bool
	SetupRequired      bool
	Username           string
	AvatarPath         string
	RecoveryKeyReady   bool
}

type Service struct {
	accountRepo *accountsetting.AccountSettingsRepository
	sessionRepo *session.SessionRepository
}

func NewService(
	accountRepo *accountsetting.AccountSettingsRepository,
	sessionRepo *session.SessionRepository,
) *Service {
	return &Service{
		accountRepo: accountRepo,
		sessionRepo: sessionRepo,
	}
}

func (s *Service) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}

			viewer, clearCookie, err := s.loadViewer(r)
			if err != nil {
				http.Error(w, "Unable to load authentication state.", http.StatusInternalServerError)
				return
			}

			if clearCookie {
				clearSessionCookie(w, r)
			}

			ctx := context.WithValue(r.Context(), viewerContextKey{}, viewer)
			r = r.WithContext(ctx)

			if isAPIPath(r.URL.Path) {
				if viewer.SetupRequired {
					http.Error(w, "Administrator setup is still required.", http.StatusServiceUnavailable)
					return
				}

				if viewer.IsAuthenticated || viewer.IsAPIAuthenticated {
					next.ServeHTTP(w, r)
					return
				}

				http.Error(w, "Unauthorized.", http.StatusUnauthorized)
				return
			}

			if target := redirectTarget(r.URL.Path, viewer); target != "" {
				redirectRequest(w, r, target)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) Account(ctx context.Context) (*accountsetting.AccountSettings, error) {
	return s.accountRepo.Get(ctx)
}

func (s *Service) Authenticate(ctx context.Context, username, password string) (*accountsetting.AccountSettings, error) {
	account, err := s.accountRepo.Get(ctx)
	if err != nil || account == nil {
		return nil, err
	}

	if strings.TrimSpace(username) != account.Username {
		return nil, nil
	}

	if !security.VerifyPassword(account.PasswordHash, password) {
		return nil, nil
	}

	return account, nil
}

func (s *Service) StartSession(w http.ResponseWriter, r *http.Request) error {
	plainToken, hashedToken, _, err := security.GenerateToken()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(sessionTTL)
	if _, err := s.sessionRepo.Create(r.Context(), hashedToken, expiresAt); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    plainToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  expiresAt,
		MaxAge:   int(sessionTTL.Seconds()),
	})

	return nil
}

func (s *Service) EndSession(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil && strings.TrimSpace(cookie.Value) != "" {
		if deleteErr := s.sessionRepo.DeleteByTokenHash(r.Context(), security.HashToken(cookie.Value)); deleteErr != nil {
			return deleteErr
		}
	}

	clearSessionCookie(w, r)
	return nil
}

func (s *Service) ClearAllSessions(ctx context.Context) error {
	return s.sessionRepo.DeleteAll(ctx)
}

func (s *Service) ResetPasswordWithRecovery(
	ctx context.Context,
	username, recoveryKey, newPassword string,
) (*accountsetting.AccountSettings, error) {
	account, err := s.accountRepo.Get(ctx)
	if err != nil {
		return nil, err
	}

	if account == nil {
		return nil, errors.New("account settings are not configured yet")
	}

	if strings.TrimSpace(username) != account.Username {
		return nil, errors.New("recovery details did not match the configured account")
	}

	if !security.RecoverySeedMatchesPublicKey(recoveryKey, account.RecoveryPublicKey) {
		return nil, errors.New("recovery key is invalid")
	}

	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return nil, err
	}

	account.PasswordHash = passwordHash
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, err
	}

	return account, nil
}

func ViewerFromContext(ctx context.Context) Viewer {
	viewer, ok := ctx.Value(viewerContextKey{}).(Viewer)
	if !ok {
		return Viewer{}
	}

	return viewer
}

func (s *Service) loadViewer(r *http.Request) (Viewer, bool, error) {
	account, err := s.accountRepo.Get(r.Context())
	if err != nil {
		return Viewer{}, false, err
	}

	if account == nil {
		return Viewer{SetupRequired: true}, false, nil
	}

	viewer := Viewer{
		HasAccount:       true,
		Username:         account.Username,
		AvatarPath:       account.AvatarPath,
		RecoveryKeyReady: strings.TrimSpace(account.RecoveryPublicKey) != "",
	}

	if s.apiTokenMatches(account, r) {
		viewer.IsAPIAuthenticated = true
	}

	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return viewer, false, nil
	}

	sessionTokenHash := security.HashToken(cookie.Value)
	record, err := s.sessionRepo.GetValidByTokenHash(r.Context(), sessionTokenHash, time.Now())
	if err != nil {
		return Viewer{}, false, err
	}

	if record == nil {
		return viewer, true, nil
	}

	viewer.IsAuthenticated = true
	return viewer, false, nil
}

func redirectTarget(path string, viewer Viewer) string {
	if viewer.SetupRequired {
		switch path {
		case "/account-settings", SetupPath, "/logout", "/javascript-license-info":
			return ""
		default:
			return SetupPath
		}
	}

	if viewer.IsAuthenticated {
		if path == LoginPath {
			return DashboardPath
		}
		return ""
	}

	switch path {
	case LoginPath, "/logout", "/javascript-license-info":
		return ""
	default:
		return LoginPath
	}
}

func redirectRequest(w http.ResponseWriter, r *http.Request, target string) {
	if htmx.IsRequest(r) {
		htmx.Redirect(w, target)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r != nil && r.TLS != nil,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (s *Service) apiTokenMatches(account *accountsetting.AccountSettings, r *http.Request) bool {
	if account == nil || r == nil || strings.TrimSpace(account.APITokenHash) == "" {
		return false
	}

	token := strings.TrimSpace(r.Header.Get("X-API-Token"))
	if token == "" {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if scheme, rest, ok := strings.Cut(authHeader, " "); ok && strings.EqualFold(scheme, "Bearer") {
			token = strings.TrimSpace(rest)
		}
	}

	if token == "" {
		return false
	}

	return security.HashToken(token) == account.APITokenHash
}

func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/")
}
