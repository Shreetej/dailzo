package utils

import (
	"dailzo/config"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret []byte

func init() {
	jwtSecret = []byte(config.LoadConfig().JWTSecret)
}

// GenerateJWT signs a token carrying both the user id and their role.
// user_type is what middleware.JWTMiddleware reads into c.Locals("user_type")
// for RoleMiddleware/AdminOnly/DeliveryOnly/etc — those checks silently never
// matched anything until now because no caller populated this claim.
func GenerateJWT(userID, userType string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":       userID,
		"user_type": userType,
		"iat":       now.Unix(),
		"exp":       now.Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseJWT(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
