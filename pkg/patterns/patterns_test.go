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

func TestCreditCard_Matches(t *testing.T) {
	p, _ := patterns.Get("credit-card")
	cases := []string{"4111111111111111", "4111 1111 1111 1111"}
	for _, c := range cases {
		if !p.Pattern.MatchString(c) {
			t.Errorf("credit-card should match %q", c)
		}
	}
}

func TestIPv4Private_Matches(t *testing.T) {
	p, _ := patterns.Get("ipv4-private")
	cases := []string{"10.0.0.1", "192.168.1.100", "172.16.0.1"}
	for _, c := range cases {
		if !p.Pattern.MatchString(c) {
			t.Errorf("ipv4-private should match %q", c)
		}
	}
}

func TestUUID_Matches(t *testing.T) {
	p, _ := patterns.Get("uuid")
	if !p.Pattern.MatchString("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("uuid should match a valid UUID v4")
	}
}

func TestAadhaar_Matches(t *testing.T) {
	p, _ := patterns.Get("aadhaar")
	if !p.Pattern.MatchString("2345 6789 0123") {
		t.Error("aadhaar should match valid 12-digit aadhaar number")
	}
}

func TestPhoneIN_NoMatch(t *testing.T) {
	p, _ := patterns.Get("phone-in")
	if p.Pattern.MatchString("5876543210") {
		t.Error("phone-in should not match number starting with 5")
	}
}

func TestIPv4Private_NoMatch(t *testing.T) {
	p, _ := patterns.Get("ipv4-private")
	if p.Pattern.MatchString("8.8.8.8") {
		t.Error("ipv4-private should not match public IP 8.8.8.8")
	}
}

func TestAadhaar_NoMatch(t *testing.T) {
	p, _ := patterns.Get("aadhaar")
	if p.Pattern.MatchString("1234 5678 9012") {
		t.Error("aadhaar should not match number starting with 1")
	}
}

func TestJWT_NoMatch(t *testing.T) {
	p, _ := patterns.Get("jwt")
	if p.Pattern.MatchString("notajwt.nope.nope") {
		t.Error("jwt should not match plain dot-separated string")
	}
}

func TestPANIN_NoMatch(t *testing.T) {
	p, _ := patterns.Get("pan-in")
	if p.Pattern.MatchString("abcde1234f") {
		t.Error("pan-in should not match lowercase string")
	}
}
