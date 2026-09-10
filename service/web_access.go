package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/slxar/rustdesk-api-goravel/v3/global"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
)

var ErrWebAccess = errors.New("access is invalid, expired, or already used")
var webPeerID = regexp.MustCompile(`^[A-Za-z0-9_-]{6,64}$`)

// WebAccess is a short-lived capability, never a RustDesk host credential.
type WebAccess struct {
	PeerID             string
	Issuer             uint
	IntegrationTokenID uint
	Expires            time.Time
	mu                 sync.Mutex
	relays             map[string]time.Time
	connections        int
}

// ponytail: one process owns ephemeral capabilities; restart revokes them. Use a
// shared atomic store before running multiple API replicas.
type WebAccessStore struct {
	mu       sync.Mutex
	links    map[[32]byte]*WebAccess
	sessions map[[32]byte]*WebAccess
}

var WebAccesses = &WebAccessStore{}

func webSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *WebAccessStore) prune(now time.Time) {
	for _, entries := range []map[[32]byte]*WebAccess{s.links, s.sessions} {
		for k, v := range entries {
			if !now.Before(v.Expires) {
				delete(entries, k)
			}
		}
	}
}

func (s *WebAccessStore) Issue(peer string, issuer uint, ttl time.Duration) (string, error) {
	return s.issue(peer, issuer, ttl, 0)
}

func (s *WebAccessStore) issue(peer string, issuer uint, ttl time.Duration, integrationTokenID uint) (string, error) {
	if !webPeerID.MatchString(peer) || issuer == 0 || ttl <= 0 || ttl > 5*time.Minute {
		return "", ErrWebAccess
	}
	token, err := webSecret()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(time.Now())
	if len(s.links)+len(s.sessions) >= 10000 {
		return "", errors.New("web access capacity reached")
	}
	if s.links == nil {
		s.links = make(map[[32]byte]*WebAccess)
	}
	s.links[sha256.Sum256([]byte(token))] = &WebAccess{PeerID: peer, Issuer: issuer, IntegrationTokenID: integrationTokenID, Expires: time.Now().Add(ttl)}
	return token, nil
}

func (s *WebAccessStore) Redeem(token string) (string, *WebAccess, error) {
	if len(token) != 43 {
		return "", nil, ErrWebAccess
	}
	cookie, err := webSecret()
	if err != nil {
		return "", nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sha256.Sum256([]byte(token))
	a := s.links[key]
	delete(s.links, key) // Atomic consumption, including concurrent requests.
	if a == nil || !time.Now().Before(a.Expires) {
		return "", nil, ErrWebAccess
	}
	a.Expires = time.Now().Add(time.Hour)
	a.relays = make(map[string]time.Time)
	if s.sessions == nil {
		s.sessions = make(map[[32]byte]*WebAccess)
	}
	s.sessions[sha256.Sum256([]byte(cookie))] = a
	return cookie, a, nil
}

func (s *WebAccessStore) Session(cookie string) *WebAccess {
	if len(cookie) != 43 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(time.Now())
	return s.sessions[sha256.Sum256([]byte(cookie))]
}

func (s *WebAccessStore) Revoke(cookie string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sha256.Sum256([]byte(cookie)))
}

func (a *WebAccess) AllowRelay(id string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for k, expiry := range a.relays {
		if time.Now().After(expiry) {
			delete(a.relays, k)
		}
	}
	if len(id) < 16 || len(id) > 128 || len(a.relays) >= 4 {
		return false
	}
	a.relays[id] = time.Now().Add(time.Minute)
	return true
}

func (a *WebAccess) ClaimRelay(id string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	expiry, ok := a.relays[id]
	delete(a.relays, id)
	return ok && time.Now().Before(expiry)
}

func (a *WebAccess) Acquire() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.connections >= 4 || !time.Now().Before(a.Expires) {
		return false
	}
	a.connections++
	return true
}

func (a *WebAccess) Release() { a.mu.Lock(); defer a.mu.Unlock(); a.connections-- }

// Sharing an address-book entry requires its existing full-control permission.
func CanShareWebPeer(user *model.User, peer string) bool {
	if user == nil || user.Id == 0 || !webPeerID.MatchString(peer) || !AllService.UserService.CheckUserEnable(user) {
		return false
	}
	if AllService.UserService.IsAdmin(user) {
		return true
	}
	p := AllService.PeerService.FindById(peer)
	if p.RowId != 0 && p.UserId == user.Id {
		return true
	}
	var entries []model.AddressBook
	if DB.Where("id = ?", peer).Find(&entries).Error != nil {
		return false
	}
	for _, entry := range entries {
		if AllService.AddressBookService.CheckUserFullControlPrivilege(user, entry.UserId, entry.CollectionId) {
			return true
		}
	}
	return false
}

func WebAccessOrigin() string {
	u, err := url.Parse(global.Config.Rustdesk.ApiServer)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return ""
	}
	return "https://" + u.Host
}

func IssueWebLink(user *model.User, peer string, ttl time.Duration) (string, error) {
	return IssueWebLinkForIntegration(user, peer, ttl, nil)
}

func IssueWebLinkForIntegration(user *model.User, peer string, ttl time.Duration, integrationToken *model.IntegrationToken) (string, error) {
	if WebAccessOrigin() == "" || !CanShareWebPeer(user, peer) {
		return "", ErrWebAccess
	}
	if strings.TrimSpace(global.Config.Rustdesk.Key) == "" {
		return "", errors.New("RustDesk server public key is required")
	}
	var source uint
	if integrationToken != nil {
		if integrationToken.IssuerID != user.Id || !IntegrationTokenAllowsTarget(integrationToken, peer) || integrationToken.RevokedAt != 0 || integrationToken.ExpiresAt <= time.Now().Unix() {
			return "", ErrWebAccess
		}
		source = integrationToken.Id
	}
	token, err := WebAccesses.issue(peer, user.Id, ttl, source)
	if err != nil {
		return "", err
	}
	return WebAccessOrigin() + "/webclient/open#" + token, nil
}

func WebAccessValid(a *WebAccess) bool {
	if a == nil {
		return false
	}
	user := AllService.UserService.InfoById(a.Issuer)
	if !CanShareWebPeer(user, a.PeerID) {
		return false
	}
	if a.IntegrationTokenID == 0 {
		return true
	}
	token, err := GetIntegrationToken(a.IntegrationTokenID)
	return err == nil && token.IssuerID == a.Issuer && token.RevokedAt == 0 && token.ExpiresAt > time.Now().Unix() && AllService.UserService.IsAdmin(user) && IntegrationTokenAllowsTarget(token, a.PeerID)
}
