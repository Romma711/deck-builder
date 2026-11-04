package httpapi

import (
	"net/http"
	"time"

	authapp "github.com/Romma711/deck-builder/internal/app/auth"
	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/pkg/validation"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/gin-gonic/gin"
)

type authHandler struct {
	svc *authapp.Service
}

// RegisterAuthRoutes monta el bloque de autenticación de la API v3:
//
//	POST /auth/register | /auth/login | /auth/refresh | /auth/logout  (públicos)
//	GET|PUT|DELETE /me                                               (autenticados)
func RegisterAuthRoutes(rg *gin.RouterGroup, svc *authapp.Service, tokens ports.TokenService) {
	h := &authHandler{svc: svc}

	public := rg.Group("/auth")
	public.POST("/register", h.register)
	public.POST("/login", h.login)
	public.POST("/refresh", h.refresh)
	public.POST("/logout", h.logout)

	me := rg.Group("/me", Auth(tokens))
	me.GET("", h.getMe)
	me.PUT("", h.updateProfile)
	me.DELETE("", h.deleteAccount)
}

// --- request/response ---

type registerRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type updateProfileRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

func newTokenResponse(p *authapp.TokenPair) tokenResponse {
	return tokenResponse{
		AccessToken:  p.AccessToken,
		RefreshToken: p.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    p.ExpiresIn,
	}
}

// --- handlers ---

func (h *authHandler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, &domain.ValidationError{Message: "JSON inválido"})
		return
	}
	if err := validation.Struct(req); err != nil {
		RespondError(c, err)
		return
	}

	user, err := h.svc.Register(c.Request.Context(), authapp.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, newUserResponse(user))
}

func (h *authHandler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, &domain.ValidationError{Message: "JSON inválido"})
		return
	}
	if err := validation.Struct(req); err != nil {
		RespondError(c, err)
		return
	}

	pair, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, newTokenResponse(pair))
}

func (h *authHandler) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		RespondError(c, &domain.ValidationError{Message: "refresh_token requerido"})
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, newTokenResponse(pair))
}

func (h *authHandler) logout(c *gin.Context) {
	var req logoutRequest
	_ = c.ShouldBindJSON(&req) // cuerpo opcional

	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *authHandler) getMe(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		RespondError(c, domain.ErrUnauthorized)
		return
	}

	account, err := h.svc.Me(c.Request.Context(), user.ID)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, newUserResponse(account))
}

func (h *authHandler) updateProfile(c *gin.Context) {
	current, ok := CurrentUser(c)
	if !ok {
		RespondError(c, domain.ErrUnauthorized)
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, &domain.ValidationError{Message: "JSON inválido"})
		return
	}
	if err := validation.Struct(req); err != nil {
		RespondError(c, err)
		return
	}

	account, err := h.svc.UpdateProfile(c.Request.Context(), current.ID, authapp.UpdateProfileInput{
		Username: req.Username,
		Email:    req.Email,
	})
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, newUserResponse(account))
}

func (h *authHandler) deleteAccount(c *gin.Context) {
	current, ok := CurrentUser(c)
	if !ok {
		RespondError(c, domain.ErrUnauthorized)
		return
	}

	if err := h.svc.DeleteAccount(c.Request.Context(), current.ID); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
