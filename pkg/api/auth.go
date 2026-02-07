package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Config contains runtime settings loaded from environment variables.
//
// Environment variables:
//   - TODO_PASSWORD: password used for login and JWT signing key
//   - TODO_PORT:     HTTP server port
//   - TODO_DBFILE:   path to SQLite database file
type Config struct {
	TodoPassword string
	TodoPort     string
	TodoDBFile   string
}

var (
	// appConfig is a singleton instance created by GetConfig().
	appConfig *Config
	// configOnce ensures config is initialized only once.
	configOnce sync.Once
)

// GetConfig returns application config loaded from environment variables.
//
// Defaults are applied if variables are not set.
func GetConfig() *Config {
	configOnce.Do(func() {
		appConfig = &Config{
			TodoPassword: os.Getenv("TODO_PASSWORD"),
			TodoPort:     os.Getenv("TODO_PORT"),
			TodoDBFile:   os.Getenv("TODO_DBFILE"),
		}

		// Default values for local development.
		if appConfig.TodoPassword == "" {
			appConfig.TodoPassword = "12345"
		}
		if appConfig.TodoPort == "" {
			appConfig.TodoPort = "7540"
		}
		if appConfig.TodoDBFile == "" {
			appConfig.TodoDBFile = "scheduler.db"
		}
	})
	return appConfig
}

// Claims describes JWT payload used by this app.
//
// PasswordHash is used to invalidate all previously issued tokens
// when the configured password changes.
type Claims struct {
	PasswordHash string `json:"pwd_hash"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT token from either:
//   - Cookie "token", or
//   - Authorization: Bearer <token>
//
// If TODO_PASSWORD is empty, authentication is considered disabled
// and requests are passed through.
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := GetConfig()
		password := config.TodoPassword

		if password == "" {
			next.ServeHTTP(w, r)
			return
		}

		var tokenString string
		cookie, err := r.Cookie("token")
		if err == nil {
			tokenString = cookie.Value
		}

		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			http.Error(w, "Требуется аутентификация", http.StatusUnauthorized)
			return
		}

		valid := validateToken(tokenString, password)
		if !valid {
			http.Error(w, "Требуется аутентификация", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// validateToken validates token signature and checks claims.
func validateToken(tokenString, currentPassword string) bool {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return []byte(currentPassword), nil
	})

	if err != nil || !token.Valid {
		return false
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims.PasswordHash == getPasswordHash(currentPassword)
	}

	return false
}

// getPasswordHash returns a hash representation stored in JWT claims.
//
// Note: currently it's a no-op (returns password as-is).
// This is enough for the учебный/portfolio project,
// but in a real system you would never store a raw password value.
func getPasswordHash(password string) string {
	return password
}

// SigninHandler authenticates user by password and returns a JWT token.
//
// Request:  POST /api/signin
// Body:     {"password": "..."}
// Response: {"token": "..."}
func SigninHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var creds struct {
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Неверный JSON", http.StatusBadRequest)
		return
	}

	config := GetConfig()
	password := config.TodoPassword

	if password == "" {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Аутентификация не настроена"})
		return
	}

	if creds.Password != password {
		respondWithJSON(w, http.StatusUnauthorized, map[string]string{"error": "Неверный пароль"})
		return
	}

	claims := &Claims{
		PasswordHash: getPasswordHash(password),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(password))
	if err != nil {
		http.Error(w, "Ошибка генерации токена", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"token": tokenString})
}

// respondWithJSON writes JSON response with the given status code.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Ошибка создания ответа", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
