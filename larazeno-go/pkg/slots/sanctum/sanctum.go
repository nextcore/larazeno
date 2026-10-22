package sanctum

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"larazeno-core/pkg/db"
)

type PersonalAccessToken struct {
	ID            int64
	TokenableType string
	TokenableID   int64
	Name          string
	TokenHash     string
	Abilities     string
	LastUsedAt    sql.NullTime
	ExpiresAt     sql.NullTime
}

type NewAccessToken struct {
	PlainTextToken string
	AccessToken    *PersonalAccessToken
}

// GenerateToken membuat token plain text acak (40 karakter hex) ala Sanctum
func GenerateToken() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken melakukan hashing SHA-256 pada token (persis seperti hash bawaan Sanctum)
func HashToken(plainToken string) string {
	hasher := sha256.New()
	hasher.Write([]byte(plainToken))
	return hex.EncodeToString(hasher.Sum(nil))
}

// CreateToken meng-issue token baru untuk user dan menyimpannya di DB
// Format return token Sanctum: "<id>|<plainTextToken>"
func CreateToken(tokenableType string, tokenableID int64, tokenName string, abilities []string) (*NewAccessToken, error) {
	plainToken, err := GenerateToken()
	if err != nil {
		return nil, err
	}

	tokenHash := HashToken(plainToken)
	abilitiesJSON := `["*"]`
	now := time.Now()

	query := `
		INSERT INTO personal_access_tokens (tokenable_type, tokenable_id, name, token, abilities, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	res, err := db.DBConn.DB.Exec(query, tokenableType, tokenableID, tokenName, tokenHash, abilitiesJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("gagal insert personal_access_token: %w", err)
	}

	insertID, _ := res.LastInsertId()
	plainWithID := fmt.Sprintf("%d|%s", insertID, plainToken)

	return &NewAccessToken{
		PlainTextToken: plainWithID,
		AccessToken: &PersonalAccessToken{
			ID:            insertID,
			TokenableType: tokenableType,
			TokenableID:   tokenableID,
			Name:          tokenName,
			TokenHash:     tokenHash,
			Abilities:     abilitiesJSON,
		},
	}, nil
}

// AuthenticateToken memverifikasi Bearer token Sanctum dan mengembalikan user id yang valid
func AuthenticateToken(bearerToken string) (*PersonalAccessToken, map[string]interface{}, error) {
	bearerToken = strings.TrimSpace(bearerToken)
	if bearerToken == "" {
		return nil, nil, fmt.Errorf("token kosong")
	}

	var tokenIDStr string
	var plainToken string

	// Sanctum token format: "<token_id>|<plain_token>"
	if strings.Contains(bearerToken, "|") {
		parts := strings.SplitN(bearerToken, "|", 2)
		tokenIDStr = parts[0]
		plainToken = parts[1]
	} else {
		plainToken = bearerToken
	}

	hash := HashToken(plainToken)

	var tokenRecord PersonalAccessToken
	var query string
	var row *sql.Row

	if tokenIDStr != "" {
		query = `SELECT id, tokenable_type, tokenable_id, name, token, abilities, last_used_at, expires_at 
		         FROM personal_access_tokens WHERE id = ? AND token = ? LIMIT 1`
		row = db.DBConn.DB.QueryRow(query, tokenIDStr, hash)
	} else {
		query = `SELECT id, tokenable_type, tokenable_id, name, token, abilities, last_used_at, expires_at 
		         FROM personal_access_tokens WHERE token = ? LIMIT 1`
		row = db.DBConn.DB.QueryRow(query, hash)
	}

	err := row.Scan(
		&tokenRecord.ID,
		&tokenRecord.TokenableType,
		&tokenRecord.TokenableID,
		&tokenRecord.Name,
		&tokenRecord.TokenHash,
		&tokenRecord.Abilities,
		&tokenRecord.LastUsedAt,
		&tokenRecord.ExpiresAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("token tidak valid atau tidak ditemukan")
	}

	// Update last_used_at secara asynchronous / non-blocking
	go func(id int64) {
		_, _ = db.DBConn.DB.Exec("UPDATE personal_access_tokens SET last_used_at = ? WHERE id = ?", time.Now(), id)
	}(tokenRecord.ID)

	// Ambil data user terkait
	userQuery := `SELECT id, name, email, created_at FROM users WHERE id = ? LIMIT 1`
	userRow := db.DBConn.DB.QueryRow(userQuery, tokenRecord.TokenableID)

	var uID int64
	var uName, uEmail, uCreatedAt string
	if err := userRow.Scan(&uID, &uName, &uEmail, &uCreatedAt); err != nil {
		return nil, nil, fmt.Errorf("user pemilik token tidak ditemukan")
	}

	userData := map[string]interface{}{
		"id":         uID,
		"name":       uName,
		"email":      uEmail,
		"created_at": uCreatedAt,
	}

	return &tokenRecord, userData, nil
}
