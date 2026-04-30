package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db *pgxpool.Pool
}

type AuthResult struct {
	APIKey string
	Plan   string
}

var ErrEmailExists = errors.New("email already registered")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUserInactive = errors.New("user inactive")
var ErrPlanNotFound = errors.New("plan not found")
var ErrInvalidInput = errors.New("invalid input")

func NewAuthService(db *pgxpool.Pool) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Register(ctx context.Context, email, password, planCode string) (AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	planCode = strings.TrimSpace(planCode)
	if planCode == "" {
		planCode = "free"
	}

	if !isValidEmail(email) || len(password) < 8 {
		return AuthResult{}, ErrInvalidInput
	}

	planID, err := s.findPlanID(ctx, planCode)
	if err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AuthResult{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var userID int64
	insertUser := `
INSERT INTO users (email, password_hash, status)
VALUES ($1, $2, 'active')
RETURNING id`

	err = tx.QueryRow(ctx, insertUser, email, string(passwordHash)).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			return AuthResult{}, ErrEmailExists
		}
		return AuthResult{}, err
	}

	insertPlan := `
INSERT INTO user_plans (user_id, plan_id, status, started_at)
VALUES ($1, $2, 'active', NOW())`
	if _, err = tx.Exec(ctx, insertPlan, userID, planID); err != nil {
		return AuthResult{}, err
	}

	apiKey, keyPrefix, keyHash, err := generateAPIKey()
	if err != nil {
		return AuthResult{}, err
	}

	insertKey := `
INSERT INTO api_keys (user_id, key_prefix, key_hash, is_active)
VALUES ($1, $2, $3, TRUE)`
	if _, err = tx.Exec(ctx, insertKey, userID, keyPrefix, keyHash); err != nil {
		return AuthResult{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{APIKey: apiKey, Plan: planCode}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (AuthResult, error) {
	_, planCode, err := s.authenticateUser(ctx, email, password)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{Plan: planCode}, nil
}

func (s *AuthService) IssueAPIKey(ctx context.Context, email, password string) (AuthResult, error) {
	userID, planCode, err := s.authenticateUser(ctx, email, password)
	if err != nil {
		return AuthResult{}, err
	}

	apiKey, keyPrefix, keyHash, err := generateAPIKey()
	if err != nil {
		return AuthResult{}, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AuthResult{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, "UPDATE api_keys SET is_active = FALSE WHERE user_id = $1 AND is_active = TRUE", userID); err != nil {
		return AuthResult{}, err
	}

	if _, err = tx.Exec(ctx, "INSERT INTO api_keys (user_id, key_prefix, key_hash, is_active) VALUES ($1, $2, $3, TRUE)", userID, keyPrefix, keyHash); err != nil {
		return AuthResult{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{APIKey: apiKey, Plan: planCode}, nil
}

func (s *AuthService) ChangePlan(ctx context.Context, email, password, planCode string) (AuthResult, error) {
	userID, _, err := s.authenticateUser(ctx, email, password)
	if err != nil {
		return AuthResult{}, err
	}

	planID, err := s.findPlanID(ctx, strings.TrimSpace(planCode))
	if err != nil {
		return AuthResult{}, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AuthResult{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, "UPDATE user_plans SET status = 'inactive', ends_at = NOW() WHERE user_id = $1 AND status = 'active'", userID); err != nil {
		return AuthResult{}, err
	}

	if _, err = tx.Exec(ctx, "INSERT INTO user_plans (user_id, plan_id, status, started_at) VALUES ($1, $2, 'active', NOW())", userID, planID); err != nil {
		return AuthResult{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{Plan: strings.TrimSpace(planCode)}, nil
}

func (s *AuthService) findPlanID(ctx context.Context, planCode string) (int, error) {
	const q = `SELECT id FROM plans WHERE code = $1`
	var planID int
	if err := s.db.QueryRow(ctx, q, planCode).Scan(&planID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrPlanNotFound
		}
		return 0, err
	}
	return planID, nil
}

func (s *AuthService) getActivePlan(ctx context.Context, userID int64) (string, error) {
	const q = `
SELECT p.code
FROM user_plans up
JOIN plans p ON p.id = up.plan_id
WHERE up.user_id = $1
  AND up.status = 'active'
  AND (up.ends_at IS NULL OR up.ends_at > NOW())
ORDER BY up.started_at DESC
LIMIT 1`

	var planCode string
	if err := s.db.QueryRow(ctx, q, userID).Scan(&planCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrPlanNotFound
		}
		return "", err
	}

	return planCode, nil
}

func (s *AuthService) authenticateUser(ctx context.Context, email, password string) (int64, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !isValidEmail(email) || password == "" {
		return 0, "", ErrInvalidInput
	}

	const userQuery = `
SELECT id, password_hash, status
FROM users
WHERE email = $1`

	var userID int64
	var passwordHash string
	var status string
	if err := s.db.QueryRow(ctx, userQuery, email).Scan(&userID, &passwordHash, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", ErrInvalidCredentials
		}
		return 0, "", err
	}

	if status != "active" {
		return 0, "", ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return 0, "", ErrInvalidCredentials
	}

	planCode, err := s.getActivePlan(ctx, userID)
	if err != nil {
		return 0, "", err
	}

	return userID, planCode, nil
}

func generateAPIKey() (string, string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", "", err
	}
	apiKey := hex.EncodeToString(buf)
	keyPrefix := apiKey
	if len(keyPrefix) > 16 {
		keyPrefix = keyPrefix[:16]
	}
	keyHash := sha256Hex(apiKey)
	return apiKey, keyPrefix, keyHash, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505"
}

func isValidEmail(value string) bool {
	return strings.Contains(value, "@") && strings.Contains(value, ".")
}
