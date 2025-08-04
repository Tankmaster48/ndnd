package ndncert

import (
	"fmt"

	"github.com/named-data/ndnd/std/types/optional"
)

type ChallengeEmail struct {
	// Email address to send the challenge to.
	Email string
	// Callback to get the code from the user.
	CodeCallback func(status string) string
}

// This function returns the constant string "email" as the identifier for the ChallengeEmail type.
func (*ChallengeEmail) Name() string {
	return KwEmail
}

// Handles the email-based challenge authentication flow by returning initial email parameters or a verification code based on the challenge status.
func (c *ChallengeEmail) Request(input ParamMap, status optional.Optional[string]) (ParamMap, error) {
	// Validate challenge configuration
	if len(c.Email) == 0 || c.CodeCallback == nil {
		return nil, fmt.Errorf("email challenge not configured")
	}

	// Initial request parameters
	if input == nil {
		return ParamMap{
			KwEmail: []byte(c.Email),
		}, nil
	}

	// Challenge response code
	if s := status.GetOr(""); s == "need-code" || s == "wrong-code" {
		code := c.CodeCallback(s)
		if code == "" {
			return nil, fmt.Errorf("no code provided")
		}

		return ParamMap{
			KwCode: []byte(code),
		}, nil
	}

	// Unknown status
	return nil, fmt.Errorf("unknown input to email challenge")
}
