// Package middleware provides HTTP interceptors and security controls.
package middleware

import (
	"net/http"
	"shopMe/internal/reuse"
	"strconv"
	"strings"
	"sync"
	"time"
)

type jailRecord struct {
	reason    string
	expiresAt time.Time
}

var (
	jailMu        sync.RWMutex
	ipJailMap     = make(map[string]jailRecord)
	stopJailPurge chan struct{}
)

func init() {
	stopJailPurge = make(chan struct{})
	// Background garbage collector to keep memory consumption near zero
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				PurgeExpiredJail()
			case <-stopJailPurge:
				return
			}
		}
	}()
}

// PurgeExpiredJail frees memory allocated to expired jail entries
func PurgeExpiredJail() {
	jailMu.Lock()
	defer jailMu.Unlock()

	now := time.Now()
	for ip, entry := range ipJailMap {
		if now.After(entry.expiresAt) {
			delete(ipJailMap, ip)
		}
	}
}

// JailIP places an offending IP address in temporary quarantine
func JailIP(ip string, duration time.Duration, reason string) {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return
	}

	jailMu.Lock()
	defer jailMu.Unlock()

	// Guard against memory exhaustion under distributed flood attacks
	if len(ipJailMap) > 20000 {
		now := time.Now()
		for k, v := range ipJailMap {
			if now.After(v.expiresAt) {
				delete(ipJailMap, k)
			}
		}
		if len(ipJailMap) > 10000 {
			ipJailMap = make(map[string]jailRecord) // Emergency fallback reset
		}
	}

	ipJailMap[ip] = jailRecord{
		reason:    reason,
		expiresAt: time.Now().Add(duration),
	}
}

// IsIPJailed checks whether an IP is quarantined and returns remaining duration
func IsIPJailed(ip string) (bool, string, int64) {
	jailMu.RLock()
	defer jailMu.RUnlock()

	rec, exists := ipJailMap[ip]
	if !exists {
		return false, "", 0
	}

	now := time.Now()
	if now.After(rec.expiresAt) {
		return false, "", 0
	}

	remainingSeconds := int64(time.Until(rec.expiresAt).Seconds())
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}

	return true, rec.reason, remainingSeconds
}

// ClearIPJail manually unjails an IP (useful for testing or administrative unbans)
func ClearIPJail(ip string) {
	jailMu.Lock()
	delete(ipJailMap, ip)
	jailMu.Unlock()
}

// Known malicious scanner User-Agent signatures
var knownMaliciousScanners = []string{
	"sqlmap",
	"nikto",
	"masscan",
	"nmap",
	"zgrab",
	"gobuster",
	"dirbuster",
	"wpscan",
	"openvas",
	"nessus",
	"acunetix",
	"havij",
	"netsparker",
	"morfeus",
	"paros",
	"arachni",
}

// Automated scraping tool user agents (blocked on mutating & sensitive routes)
var automatedScraperUAs = []string{
	"python-requests",
	"python-urllib",
	"aiohttp",
	"scrapy",
	"libwww-perl",
	"go-http-client",
	"httpx",
}

// Exploit probe path patterns that trigger instant IP jail
var exploitProbePaths = []string{
	"/.env",
	"/.git",
	"/wp-admin",
	"/wp-login.php",
	"/wp-content",
	"/wp-includes",
	"/xmlrpc.php",
	"/phpmyadmin",
	"/pma",
	"/.aws",
	"/actuator",
	"/solr",
	"/cgi-bin",
	"/shell.php",
	"/eval-stdin.php",
	"/boaform",
	"/vendor/phpunit",
	"/dump.sql",
	"/backup.sql",
	"/config.json",
	"/.ds_store",
}

// isExploitProbe checks if incoming URL path matches common vulnerability probes
func isExploitProbe(path string) bool {
	lower := strings.ToLower(path)
	for _, probe := range exploitProbePaths {
		if strings.HasPrefix(lower, probe) || strings.Contains(lower, probe) {
			return true
		}
	}
	return false
}

// isMaliciousScannerUA checks if User-Agent matches aggressive security testing tools
func isMaliciousScannerUA(ua string) bool {
	if ua == "" {
		return false
	}
	lower := strings.ToLower(ua)
	for _, scanner := range knownMaliciousScanners {
		if strings.Contains(lower, scanner) {
			return true
		}
	}
	return false
}

// isAutomatedScraperUA checks if User-Agent matches automated scripting frameworks
func isAutomatedScraperUA(ua string) bool {
	if ua == "" {
		return false
	}
	lower := strings.ToLower(ua)
	for _, scraper := range automatedScraperUAs {
		if strings.Contains(lower, scraper) {
			return true
		}
	}
	return false
}

// ValidateHoneypot inspects hidden bot trap fields.
// If populated, automatically quarantines the client IP and returns false.
func ValidateHoneypot(honeypotVal string, ip string) bool {
	if strings.TrimSpace(honeypotVal) != "" {
		JailIP(ip, 30*time.Minute, "honeypot trap triggered")
		return false
	}
	return true
}

// BotGuard inspects requests for automated malicious bots, exploit scanners, and quarantined IPs.
// Legitimate browsers, mobile apps, and search engines pass with 0ms latency.
func BotGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ExtractIP(r)

		// 1. Instant quarantine check (0 DB queries, O(1) in-memory lookup)
		if jailed, reason, retryAfter := IsIPJailed(ip); jailed {
			w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
			reuse.Error(w, http.StatusForbidden, "Access temporarily suspended due to security violations: "+reason)
			return
		}

		// 2. Exploit path probing detection (Instant 30-minute quarantine)
		if isExploitProbe(r.URL.Path) {
			JailIP(ip, 30*time.Minute, "exploit probe: "+r.URL.Path)
			reuse.Error(w, http.StatusForbidden, "Access denied: Malicious vulnerability probe detected.")
			return
		}

		ua := r.UserAgent()

		// 3. Known malicious security scanner detection
		if isMaliciousScannerUA(ua) {
			JailIP(ip, 15*time.Minute, "malicious scanner signature: "+ua)
			reuse.Error(w, http.StatusForbidden, "Access denied: Automated security scanner blocked.")
			return
		}

		// 4. Block automated scrapers and bots on sensitive mutating operations
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			// Require User-Agent on mutating operations
			if strings.TrimSpace(ua) == "" {
				reuse.Error(w, http.StatusBadRequest, "Invalid request: User-Agent header is required.")
				return
			}

			// Block automated scripts (curl, python-requests, etc.) on auth / write endpoints
			if isAutomatedScraperUA(ua) {
				reuse.Error(w, http.StatusForbidden, "Automated script client not permitted on this endpoint.")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
