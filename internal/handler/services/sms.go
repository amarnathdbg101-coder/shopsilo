package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// SMSService defines the contract for dispatching notifications and OTPs via SMS and WhatsApp.
type SMSService interface {
	SendRegistrationOTP(ctx context.Context, phone, otp string) error
	SendRegistrationWhatsAppOTP(ctx context.Context, phone, otp string) error
}

// DefaultSMSService implements SMSService with support for Meta WhatsApp Cloud API,
// SMS gateways (e.g. Fast2SMS/Twilio), and seamless development fallbacks.
type DefaultSMSService struct {
	httpClient *http.Client
}

func NewDefaultSMSService() *DefaultSMSService {
	return &DefaultSMSService{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendRegistrationWhatsAppOTP dispatches the OTP via official Meta WhatsApp Cloud API.
func (s *DefaultSMSService) SendRegistrationWhatsAppOTP(ctx context.Context, phone, otp string) error {
	token := os.Getenv("WHATSAPP_ACCESS_TOKEN")
	phoneID := os.Getenv("WHATSAPP_PHONE_NUMBER_ID")

	// Format recipient phone number for WhatsApp (e.g. 919876543210)
	waRecipient := strings.TrimSpace(phone)
	if len(waRecipient) == 10 {
		waRecipient = "91" + waRecipient
	}

	if token == "" || phoneID == "" {
		// Development fallback: Log OTP to console so development and QA can proceed without paid Meta setup
		log.Printf("[WHATSAPP/DEV] Registration OTP for +%s: %s (Valid for 5 minutes). Delivered via WhatsApp!", waRecipient, otp)
		return nil
	}

	templateName := os.Getenv("WHATSAPP_OTP_TEMPLATE")
	var payload map[string]interface{}

	if templateName != "" {
		// Official Meta Authentication Template format
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                waRecipient,
			"type":              "template",
			"template": map[string]interface{}{
				"name": templateName,
				"language": map[string]string{
					"code": "en",
				},
				"components": []map[string]interface{}{
					{
						"type": "body",
						"parameters": []map[string]string{
							{"type": "text", "text": otp},
						},
					},
					{
						"type":     "button",
						"sub_type": "url",
						"index":    "0",
						"parameters": []map[string]string{
							{"type": "text", "text": otp},
						},
					},
				},
			},
		}
	} else {
		// Standard WhatsApp message format
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                waRecipient,
			"type":              "text",
			"text": map[string]interface{}{
				"preview_url": false,
				"body":        fmt.Sprintf("Your Shopsilo registration OTP is %s. Valid for 5 minutes. Do not share it with anyone.", otp),
			},
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal WhatsApp payload: %w", err)
	}

	apiURL := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", phoneID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create WhatsApp request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[WHATSAPP/ERROR] Failed to send WhatsApp OTP: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[WHATSAPP/ERROR] WhatsApp API returned status %d: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("whatsapp api error status: %d", resp.StatusCode)
	}

	log.Printf("[WHATSAPP/SUCCESS] OTP dispatched to +%s via WhatsApp Cloud API", waRecipient)
	return nil
}

// SendRegistrationOTP dispatches the OTP via standard SMS gateway.
func (s *DefaultSMSService) SendRegistrationOTP(ctx context.Context, phone, otp string) error {
	apiKey := os.Getenv("SMS_API_KEY")
	if apiKey == "" {
		// Development fallback
		log.Printf("[SMS/DEV] Registration OTP for %s: %s (Valid for 5 minutes)", phone, otp)
		return nil
	}

	log.Printf("[SMS/GATEWAY] Dispatching OTP to %s via configured SMS gateway", phone)
	return s.dispatchExternalSMS(ctx, phone, fmt.Sprintf("Your Shopsilo registration OTP is %s. Valid for 5 minutes. Do not share it with anyone.", otp))
}

func (s *DefaultSMSService) dispatchExternalSMS(ctx context.Context, phone, message string) error {
	fast2smsURL := os.Getenv("FAST2SMS_URL")
	if fast2smsURL == "" {
		fast2smsURL = "https://www.fast2sms.com/dev/bulkV2"
	}
	apiKey := os.Getenv("SMS_API_KEY")

	// Fast2SMS Quick Route format
	endpoint := fmt.Sprintf("%s?authorization=%s&route=q&message=%s&language=english&flash=0&numbers=%s",
		fast2smsURL, apiKey, url.QueryEscape(message), phone)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
