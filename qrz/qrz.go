package qrz

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const apiURL = "https://xmldata.qrz.com/xml/current/"

type Client struct {
	username string
	password string
	mu       sync.Mutex
	session  string
	expiry   time.Time
}

func NewClient(username, password string) *Client {
	return &Client{username: username, password: password}
}

type Result struct {
	Callsign string `json:"callsign"`
	Name     string `json:"name"`
	FName    string `json:"fname"`
	LName    string `json:"lname"`
	City     string `json:"city,omitempty"`
	State    string `json:"state,omitempty"`
	Country  string `json:"country,omitempty"`
	Grid     string `json:"grid,omitempty"`
	Lat      string `json:"lat,omitempty"`
	Lon      string `json:"lon,omitempty"`
	AltM     string `json:"altm,omitempty"`
}

type qrzResponse struct {
	XMLName  xml.Name    `xml:"QRZDatabase"`
	Session  qrzSession  `xml:"Session"`
	Callsign qrzCallsign `xml:"Callsign"`
}

type qrzSession struct {
	Key   string `xml:"Key"`
	Error string `xml:"Error"`
}

type qrzCallsign struct {
	Call    string `xml:"call"`
	FName   string `xml:"fname"`
	Name    string `xml:"name"`
	Addr2   string `xml:"addr2"`
	State   string `xml:"state"`
	Country string `xml:"country"`
	Grid    string `xml:"grid"`
	Lat     string `xml:"lat"`
	Lon     string `xml:"lon"`
	AltM    string `xml:"altm"`
}

func (c *Client) getSession() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != "" && time.Now().Before(c.expiry) {
		return c.session, nil
	}

	url := apiURL + "?username=" + url.QueryEscape(c.username) + ";password=" + url.QueryEscape(c.password) + ";agent=qrzlook-1.0"
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading auth response: %w", err)
	}

	var r qrzResponse
	if err := xml.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("parsing auth response: %w", err)
	}
	if r.Session.Error != "" {
		return "", fmt.Errorf("QRZ auth error: %s", r.Session.Error)
	}
	if r.Session.Key == "" {
		return "", fmt.Errorf("no session key returned")
	}

	c.session = r.Session.Key
	c.expiry = time.Now().Add(30 * time.Minute)
	return c.session, nil
}

func (c *Client) Lookup(callsign string) (*Result, error) {
	session, err := c.getSession()
	if err != nil {
		return nil, err
	}

	result, err := c.doLookup(session, callsign)
	if err != nil && strings.Contains(err.Error(), "Invalid session") {
		c.mu.Lock()
		c.session = ""
		c.mu.Unlock()
		session, err = c.getSession()
		if err != nil {
			return nil, err
		}
		result, err = c.doLookup(session, callsign)
	}
	return result, err
}

func (c *Client) doLookup(session, callsign string) (*Result, error) {
	url := apiURL + "?s=" + session + ";callsign=" + strings.ToUpper(callsign)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("lookup request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var r qrzResponse
	if err := xml.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	if r.Session.Error != "" {
		return nil, fmt.Errorf("QRZ error: %s", r.Session.Error)
	}

	cs := r.Callsign
	if cs.Call == "" {
		return nil, fmt.Errorf("callsign not found")
	}

	displayName := cs.FName
	if cs.Name != "" {
		if displayName != "" {
			displayName += " " + cs.Name
		} else {
			displayName = cs.Name
		}
	}

	return &Result{
		Callsign: cs.Call,
		Name:     displayName,
		FName:    cs.FName,
		LName:    cs.Name,
		City:     cs.Addr2,
		State:    cs.State,
		Country:  cs.Country,
		Grid:     cs.Grid,
		Lat:      cs.Lat,
		Lon:      cs.Lon,
		AltM:     cs.AltM,
	}, nil
}
