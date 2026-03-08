package handler

type AuthRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}
