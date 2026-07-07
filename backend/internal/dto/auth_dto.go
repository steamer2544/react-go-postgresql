// Package dto defines request and response data transfer objects.
package dto

// LoginRequest is the payload for the login endpoint.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is the success payload for the login endpoint.
type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

// MeResponse is the success payload for the get-me endpoint.
type MeResponse struct {
	ID                 uint    `json:"id"`
	Email              string  `json:"email"`
	Role               string  `json:"role"`
	FullName           string  `json:"full_name"`
	Position           string  `json:"position"`
	SignatureImagePath *string `json:"signature_image_path"`
}

// UpdateProfileRequest is the payload for updating a user profile.
type UpdateProfileRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Position string `json:"position" binding:"required"`
}
