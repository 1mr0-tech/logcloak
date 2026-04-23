package patterns_test

import (
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/patterns"
)

func TestLibrary_AllBuiltinsPresent(t *testing.T) {
	required := []string{
		"email", "phone-e164", "phone-us", "otp-6digit",
		"credit-card", "jwt", "ipv4-private", "uuid", "iban", "ssn",
	}
	for _, name := range required {
		if _, ok := patterns.Get(name); !ok {
			t.Errorf("missing built-in pattern %q", name)
		}
	}
}

func TestEmail_Matches(t *testing.T) {
	p, _ := patterns.Get("email")
	cases := []string{"user@example.com", "a.b+c@foo.co.uk"}
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

func TestPhoneE164_Matches(t *testing.T) {
	p, _ := patterns.Get("phone-e164")
	cases := []string{"+12025550104", "+442071838750", "+819012345678"}
	for _, c := range cases {
		if !p.Pattern.MatchString(c) {
			t.Errorf("phone-e164 should match %q", c)
		}
	}
}

func TestPhoneE164_NoMatch(t *testing.T) {
	p, _ := patterns.Get("phone-e164")
	if p.Pattern.MatchString("notaphone") {
		t.Error("phone-e164 should not match plain string")
	}
}

func TestPhoneUS_Matches(t *testing.T) {
	p, _ := patterns.Get("phone-us")
	cases := []string{"202-555-0104", "(202) 555-0104", "+12025550104"}
	for _, c := range cases {
		if !p.Pattern.MatchString(c) {
			t.Errorf("phone-us should match %q", c)
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

func TestJWT_NoMatch(t *testing.T) {
	p, _ := patterns.Get("jwt")
	if p.Pattern.MatchString("notajwt.nope.nope") {
		t.Error("jwt should not match plain dot-separated string")
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

func TestIPv4Private_NoMatch(t *testing.T) {
	p, _ := patterns.Get("ipv4-private")
	if p.Pattern.MatchString("8.8.8.8") {
		t.Error("ipv4-private should not match public IP 8.8.8.8")
	}
}

func TestUUID_Matches(t *testing.T) {
	p, _ := patterns.Get("uuid")
	if !p.Pattern.MatchString("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("uuid should match a valid UUID v4")
	}
}

func TestIBAN_Matches(t *testing.T) {
	p, _ := patterns.Get("iban")
	cases := []string{"GB29NWBK60161331926819", "DE89370400440532013000", "FR7614508590005808498637X2"}
	for _, c := range cases {
		if !p.Pattern.MatchString(c) {
			t.Errorf("iban should match %q", c)
		}
	}
}

func TestIBAN_NoMatch(t *testing.T) {
	p, _ := patterns.Get("iban")
	if p.Pattern.MatchString("notaniban") {
		t.Error("iban should not match plain string")
	}
}

func TestSSN_Matches(t *testing.T) {
	p, _ := patterns.Get("ssn")
	cases := []string{"123-45-6789", "123 45 6789"}
	for _, c := range cases {
		if !p.Pattern.MatchString(c) {
			t.Errorf("ssn should match %q", c)
		}
	}
}

func TestSSN_NoMatch(t *testing.T) {
	p, _ := patterns.Get("ssn")
	if p.Pattern.MatchString("1234567890") {
		t.Error("ssn should not match unformatted number")
	}
}
