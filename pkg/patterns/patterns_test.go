package patterns_test

import (
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/patterns"
)

func TestLibrary_AllBuiltinsPresent(t *testing.T) {
	required := []string{
		"email", "phone-in", "phone-us", "otp-6digit",
		"credit-card", "jwt", "ipv4-private", "uuid", "aadhaar", "pan-in",
	}
	for _, name := range required {
		if _, ok := patterns.Get(name); !ok {
			t.Errorf("missing built-in pattern %q", name)
		}
	}
}

func TestEmail_Matches(t *testing.T) {
	p, _ := patterns.Get("email")
	cases := []string{"user@example.com", "a.b+c@foo.co.in"}
	for _, c := range cases {
		if !p.Pattern.MatchString(c) {
			t.Errorf("email pattern should match %q", c)
		}
	}
}

func TestEmail_NoMatch(t *testing.T) {
	p, _ := patterns.Get("email")
	if p.Pattern.MatchString("notanemail") {
		t.Error("'notanemail' should not match email pattern")
	}
}

func TestOTP6_Matches(t *testing.T) {
	p, _ := patterns.Get("otp-6digit")
	if !p.Pattern.MatchString("Your OTP is 123456 valid") {
		t.Error("otp-6digit should match 6-digit standalone number")
	}
}

func TestOTP6_NoMatchLonger(t *testing.T) {
	p, _ := patterns.Get("otp-6digit")
	if p.Pattern.MatchString("1234567") {
		t.Error("otp-6digit should not match 7-digit number")
	}
}

func TestPhoneIN_Matches(t *testing.T) {
	p, _ := patterns.Get("phone-in")
	cases := []string{"9876543210", "+919876543210", "+91-9876543210"}
	for _, c := range cases {
		if !p.Pattern.MatchString(c) {
			t.Errorf("phone-in should match %q", c)
		}
	}
}

func TestJWT_Matches(t *testing.T) {
	p, _ := patterns.Get("jwt")
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	if !p.Pattern.MatchString(token) {
		t.Error("jwt pattern should match a real JWT")
	}
}

func TestPANIN_Matches(t *testing.T) {
	p, _ := patterns.Get("pan-in")
	if !p.Pattern.MatchString("ABCDE1234F") {
		t.Error("pan-in should match valid PAN format")
	}
}
