package middleware

import "github.com/gin-gonic/gin"

const clientProfileContextKey = "client_profile"

func setClientProfile(c *gin.Context, profile ClientProfile) {
	c.Set(clientProfileContextKey, profile)
}

func getClientProfile(c *gin.Context) (ClientProfile, bool) {
	v, ok := c.Get(clientProfileContextKey)
	if !ok {
		return ClientProfile{}, false
	}

	profile, ok := v.(ClientProfile)
	return profile, ok
}
