package tencentcloud

// Option Used to mutate client
type Option func(*Client)

// Mutate method mutate client properties
func (c *Client) Mutate(option Option) {
	option(c)
}

// WithSecret mutate client secret id and secret key
func WithSecret(id, key string) Option {
	return func(c *Client) {
		c.secretID = id
		c.secretKey = key
	}
}

func WithRegion(region string) Option {
	return func(c *Client) {
		c.region = region
	}
}
