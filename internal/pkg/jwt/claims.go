package gojwt

import (
	"encoding/json"
	"time"
)

// Claims represents standard and custom JWT claims.
type Claims struct {
	Subject   string         `json:"sub,omitempty"`
	Issuer    string         `json:"iss,omitempty"`
	Audience  string         `json:"aud,omitempty"`
	ExpiresAt int64          `json:"exp,omitempty"`
	NotBefore int64          `json:"nbf,omitempty"`
	IssuedAt  int64          `json:"iat,omitempty"`
	ID        string         `json:"jti,omitempty"`
	Custom    map[string]any `json:"custom,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling to automatically capture any extra top-level claims into Custom.
func (c *Claims) UnmarshalJSON(data []byte) error {
	type Alias Claims
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if c.Custom == nil {
		c.Custom = make(map[string]any)
	}

	standardKeys := map[string]struct{}{
		"sub": {}, "iss": {}, "aud": {}, "exp": {}, "nbf": {}, "iat": {}, "jti": {}, "custom": {},
	}

	for k, v := range raw {
		if _, isStd := standardKeys[k]; !isStd {
			c.Custom[k] = v
		}
	}
	return nil
}

// Get returns a custom claim value if present.
func (c *Claims) Get(key string) (any, bool) {
	if c.Custom == nil {
		return nil, false
	}
	v, ok := c.Custom[key]
	return v, ok
}

// GetString returns a custom claim string value, or empty string.
func (c *Claims) GetString(key string) string {
	if v, ok := c.Get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Valid verifies expiration and not-before timestamps with optional clock skew leeway.
func (c *Claims) Valid(leeway time.Duration) error {
	now := time.Now().Unix()
	leewaySec := int64(leeway.Seconds())

	if c.ExpiresAt > 0 && now-leewaySec > c.ExpiresAt {
		return ErrTokenExpired
	}

	if c.NotBefore > 0 && now+leewaySec < c.NotBefore {
		return ErrTokenNotValidYet
	}

	return nil
}

// ToMap serializes claims into a map.
func (c *Claims) ToMap() (map[string]any, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var res map[string]any
	err = json.Unmarshal(data, &res)
	return res, err
}
