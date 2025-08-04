package handlers

import (
	// "context"
	"net/http"
	"time"

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

// swagger:operation POST /auth/signup auth signUp
//
// Создает нового пользователя
//
// Принимает email и пароль для регистрации пользователя.
//
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: body
//   in: body
//   description: Email и пароль для регистрации
//   required: true
//   schema:
//     type: object
//     required:
//       - email
//       - password
//     properties:
//       email:
//         type: string
//         format: email
//         example: user@example.com
//       password:
//         type: string
//         format: password
//         example: strongpassword123
// responses:
//   '200':
//     description: Пользователь успешно создан
//     schema:
//       type: object
//       properties:
//         access_token:
//           type: string
//           example: "short_life_access_token"
//         refresh_token:
//           type: string
//           example: "long_life_refresh_token"
//   '400':
//     description: Некорректный запрос (например, отсутствуют поля email или password)
//   '500':
//     description: Внутренняя ошибка сервера
func (h *AuthHandler) SignUp(c *gin.Context) {
	var req SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Проверка: существует ли пользователь с таким email
	existingUser, err := h.userService.GetUserByEmail(req.Email)
	if err == nil && existingUser != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is already in use. Please choose another one."})
		return
	}

	// Хеширование пароля
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte("diduda" + req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password hashing failed"})
		return
	}

	// Создание нового пользователя и получение user_id
	userID, err := h.userService.CreateUser(req.Email, string(hashedPwd))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Генерация токенов с user_id и ролями
	roles := []string{"V01"}
	accessToken, err := utils.GenerateAccessToken(userID, roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}
	refreshToken, err := utils.GenerateRefreshToken(userID, roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Сохранение refresh токена в Redis
	err = h.redisClient.Set("refresh:"+userID, refreshToken, 7*24*time.Hour).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store refresh token"})
		return
	}

	// Ответ клиенту
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// swagger:operation POST /auth/verify auth verify
// Verify: принимает verify token, делает verified=true
// ---
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '400':
//         description: Verification error
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
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("diduda" + req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// if !user.Verified {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Email not verified"})
	// 	return
	// }
	roles := []string{"V01"}
	at, _ := utils.GenerateAccessToken(user.ID.(string), roles)
	rt, _ := utils.GenerateRefreshToken(user.ID.(string), roles)
	h.redisClient.Set("refresh:"+user.ID.(string), rt, 7*24*time.Hour)
	c.JSON(http.StatusOK, gin.H{"access_token": at, "refresh_token": rt})
}

// swagger:operation POST /auth/refresh auth refresh
// Refresh: проверяет refresh в Redis, выдаёт новую пару
// ---
// security:
// - BearerAuth: []
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
	roles := []string{"V01"}
	at, _ := utils.GenerateAccessToken(claims.UserID, roles)
	rt, _ := utils.GenerateRefreshToken(claims.UserID, roles)
	h.redisClient.Set("refresh:"+claims.UserID, rt, 7*24*time.Hour)
	c.JSON(http.StatusOK, gin.H{"access_token": at, "refresh_token": rt})
}

// swagger:operation POST /auth/logout auth logout
// Logout: удаляет refresh токен из Redis, требует Authorization заголовок с Bearer токеном
// ---
// security:
// - BearerAuth: []
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful logout
//     '401':
//         description: Unauthorized (token missing or invalid)
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, _ := c.Get("userID")
	h.redisClient.Del("refresh:" + userID.(string))
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// swagger:operation POST /auth/me auth me
// Me: сохраняет и возвращает сессию
// ---
// produces:
// - application/json
// responses:
//     '200':
//         description: Current session information
//         schema:
//             type: object
//             properties:
//                 userID:
//                     type: string
//                 ip:
//                     type: string
//                 userAgent:
//                     type: string
//                 location:
//                     type: string
//     '401':
//         description: Unauthorized (token missing or invalid)
func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("userID")
	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	loc := "Unknown"
	session := models.SessionInfo{UserID: userID.(string), IP: ip, UserAgent: ua, Location: loc}
	models.SessionStore = append(models.SessionStore, session)
	c.JSON(http.StatusOK, session)
}
