package handlers

import (
	// "context"
	"net/http"
	"time"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"

	// "go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"

	"srv_recipes/models"
	"srv_recipes/services" // services/user.go : CreateUser, GetUserByEmail, VerifyUser
	"srv_recipes/utils"
)

// AuthHandler: обработчик аутентификации
type AuthHandler struct {
	userService *services.UserHandler
	redisClient *redis.Client
}

func NewAuthHandler(userService *services.UserHandler, redisClient *redis.Client) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		redisClient: redisClient,
	}
}

// SignUpRequest represents the request body for user signup
// swagger:model
type SignUpRequest struct {
	// User's email address
	// required: true
	// example: user@example.com
	Email string `json:"Email" binding:"required,email"`

	// User's password
	// required: true
	// example: strongpassword
	Password string `json:"Password" binding:"required,min=6"`
}

// swagger:parameters signupUser
type SignUpParams struct {
	// User signup credentials
	// in: body
	// required: true
	Body SignUpRequest `json:"body"`
}

// SignUpResponse represents successful signup response
// swagger:model
type SignUpResponse struct {
	// Verification token
	// example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
	Token string `json:"token"`
}

// ErrorResponse represents error response
// swagger:model
type ErrorResponse struct {
	// Error message
	// example: Invalid request
	Error string `json:"error"`
}

// swagger:operation POST /auth/signup auth signup
//
// # SignUp создаёт пользователя, возвращает verify token
//
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
//   - name: body
//     in: body
//     description: User signup credentials
//     required: true
//     schema:
//     "$ref": "#/definitions/SignUpRequest"
//
// responses:
//	'200':
//	  description: Successfully registered user
//	  schema:
//	    type: object
//	    properties:
//	      token:
//	        type: string
//	        description: Verification token
//	'400':
//	  description: Invalid request
//	  schema:
//	    type: object
//	    properties:
//	      error:
//	        type: string
func (h *AuthHandler) SignUp(c *gin.Context) {
	var req SignUpRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	log.Println("SignUp request received:", req)
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	log.Println("Hashed password:", string(hash))
	if err := h.userService.CreateUser(req.Email, string(hash)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	verifyToken, _ := utils.GenerateAccessToken(req.Email)
	accessToken, _ := utils.GenerateAccessToken(req.Email)
	refreshToken, _ := utils.GenerateRefreshToken(req.Email)
	c.JSON(http.StatusOK, gin.H{
		"message":      "User created. Verify email.",
		"verify_token": verifyToken,
		"access_token": accessToken,
		"refresh_token": refreshToken,
	})
}

// Verify: принимает verify token, делает verified=true
func (h *AuthHandler) Verify(c *gin.Context) {
	var req struct{ VerifyToken string }
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing verify_token"})
		return
	}
	claims, err := utils.ParseAccessToken(req.VerifyToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	if err = h.userService.VerifyUser(claims.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Email verified"})
}

// swagger:operation POST /auth/signin auth signIn
// Login: возвращает access+refresh, сохраняет refresh в Redis
// ---
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '401':
//         description: Invalid credentials
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct{ Email, Password string }
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	user, err := h.userService.GetUserByEmail(req.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if !user.Verified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email not verified"})
		return
	}
	at, _ := utils.GenerateAccessToken(user.Email)
	rt, _ := utils.GenerateRefreshToken(user.Email)
	h.redisClient.Set("refresh:"+user.Email, rt, 7*24*time.Hour)
	c.JSON(http.StatusOK, gin.H{"access_token": at, "refresh_token": rt})
}

// swagger:operation POST /auth/refresh auth refresh
// Refresh: проверяет refresh в Redis, выдаёт новую пару
// ---
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '400':
//         description: Token is new and doesn't need
//                      a refresh
//     '401':
//         description: Invalid credentials
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct{ RefreshToken string }
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing token"})
		return
	}
	claims, err := utils.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	stored, err := h.redisClient.Get("refresh:" + claims.UserID).Result()
	if err != nil || stored != req.RefreshToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token mismatch"})
		return
	}
	at, _ := utils.GenerateAccessToken(claims.UserID)
	rt, _ := utils.GenerateRefreshToken(claims.UserID)
	h.redisClient.Set("refresh:"+claims.UserID, rt, 7*24*time.Hour)
	c.JSON(http.StatusOK, gin.H{"access_token": at, "refresh_token": rt})
}

// Logout: удаляет refresh токен из Redis
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, _ := c.Get("userID")
	h.redisClient.Del("refresh:" + userID.(string))
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// Me: сохраняет и возвращает сессию
func Me(c *gin.Context) {
	userID, _ := c.Get("userID")
	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	loc := "Unknown"
	session := models.SessionInfo{UserID: userID.(string), IP: ip, UserAgent: ua, Location: loc}
	models.SessionStore = append(models.SessionStore, session)
	c.JSON(http.StatusOK, session)
}
