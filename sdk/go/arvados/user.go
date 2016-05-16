package arvados

// User is an arvados#user record
type User struct {
	UUID     string `json:"uuid,omitempty"`
	IsActive bool   `json:"is_active"`
	IsAdmin  bool   `json:"is_admin"`
	Username string `json:"username,omitempty"`
}

// CurrentUser calls arvados.v1.users.current, and returns the User
// record corresponding to this client's credentials.
func (c *Client) CurrentUser() (User, error) {
	var u User
	err := c.RequestAndDecode(&u, "GET", "arvados/v1/users/current", nil, nil)
	return u, err
}
