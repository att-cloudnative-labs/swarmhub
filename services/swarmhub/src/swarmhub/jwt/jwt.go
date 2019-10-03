package jwt

import (
	"fmt"
	"net/http"
	"os"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

const (
	RolePowerUser = 5
	RoleReadOnly  = 1
	RoleNone      = 0
)

var JwtSigningKey []byte

// Claims will be used to encode the JWT
type Claims struct {
	Username string `json:"username"`
	Role     int    `json:"role"`
	jwt.StandardClaims
}

func init() {
	JwtSigningKey = []byte(os.Getenv("JWTSIGNINGKEY"))
}

func ValidateToken(tokenString string) (bool, error) {

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return JwtSigningKey, nil
	})
	if err != nil {
		err = fmt.Errorf("while performing ValidateToken was unable to parse token: %v", err)
		return false, err
	}

	if token.Valid {
		return true, nil
	}
	return false, nil
}

func TokenRole(tokenString string) (int, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return JwtSigningKey, nil
	})
	if err != nil {
		err = fmt.Errorf("while performing TokenRole was unable to parse token: %v", err)
		return RoleNone, err
	}
	if !token.Valid {
		return RoleNone, fmt.Errorf("not a valid token")
	}

	return claims.Role, err
}

func TokenAudienceFromRequest(r *http.Request) string {

	cookie, err := r.Cookie("Authorization")
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}

	audience, err := tokenAudience(cookie.Value)
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}

	return audience
}

func tokenAudience(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return JwtSigningKey, nil
	})
	if err != nil {
		err = fmt.Errorf("failed to ParseWithClaims for tokenAudience: %v", err)
		return "", err
	}
	if !token.Valid {
		err = fmt.Errorf("while extracting username the token was not valid: %v", err)
		return "", err
	}

	return claims.Username, err
}

func decryptToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return JwtSigningKey, nil
	})
	if err != nil {
		err = fmt.Errorf("failed to ParseWithClaims while decrypting: %v", err)
		return claims, err
	}

	if !token.Valid {
		return claims, fmt.Errorf("token was not valid")
	}
	return claims, nil
}

func decryptTokenFromRequest(r *http.Request) (*Claims, error) {
	cookie, err := r.Cookie("Authorization")
	if err != nil {
		return &Claims{}, err
	}
	token, err := decryptToken(cookie.Value)
	return token, err
}

// CreateToken creates a new JWT token.
func CreateToken(username string, level int) (string, error) {
	expireTime := time.Now().Add(time.Hour * 24).Unix()
	claims := &Claims{
		Username: username,
		Role:     level,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expireTime,
			Issuer:    "swarmhub",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedString, err := token.SignedString(JwtSigningKey)
	if err != nil {
		err = fmt.Errorf("failed to sign the token: %v", err)
		fmt.Println(err)
		return "", err
	}

	return signedString, nil
}
