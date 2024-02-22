package group

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
	"sort"
)

func TestGroup(t *testing.T) {
	groups.groups = nil
	Add("group", &Description{})
	Add("group/subgroup", &Description{Public: true})
	if len(groups.groups) != 2 {
		t.Errorf("Expected 2, got %v", len(groups.groups))
	}
	g := Get("group")
	g2 := Get("group/subgroup")
	if g == nil {
		t.Fatalf("Couldn't get group")
	}
	if g2 == nil {
		t.Fatalf("Couldn't get group/subgroup")
	}
	if name := g.Name(); name != "group" {
		t.Errorf("Name: expected group1, got %v", name)
	}
	if locked, _ := g.Locked(); locked {
		t.Errorf("Locked: expected false, got %v", locked)
	}
	api, err := g.API()
	if err != nil || api == nil {
		t.Errorf("Couldn't get API: %v", err)
	}

	if names := GetNames(); len(names) != 2 {
		t.Errorf("Expected 2, got %v", names)
	}

	if subs := GetSubGroups("group"); len(subs) != 0 {
		t.Errorf("Expected [], got %v", subs)
	}

	if public := GetPublic(""); len(public) != 1 || public[0].Name != "group/subgroup" {
		t.Errorf("Expected group/subgroup, got %v", public)
	}
}

func TestChatHistory(t *testing.T) {
	g := Group{
		description: &Description{},
	}
	user := "user"
	for i := 0; i < 2*maxChatHistory; i++ {
		g.AddToChatHistory("id", &user, time.Now(), "",
			fmt.Sprintf("%v", i),
		)
	}
	h := g.GetChatHistory()
	if len(h) != maxChatHistory {
		t.Errorf("Expected %v, got %v", maxChatHistory, len(g.history))
	}
	for i, s := range h {
		e := fmt.Sprintf("%v", i+maxChatHistory)
		if s.Value.(string) != e {
			t.Errorf("Expected %v, got %v", e, s)
		}
	}
}

func permissionsEqual(a, b []string) bool {
	// nil case
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	aa := append([]string(nil), a...)
	sort.Slice(aa, func(i, j int) bool {
		return aa[i] < aa[j]
	})
	bb := append([]string(nil), b...)
	sort.Slice(bb, func(i, j int) bool {
		return bb[i] < bb[j]
	})
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}

var descJSON = `
{
    "max-history-age": 10,
    "allow-subgroups": true,
    "users": {
        "jch": {"password": "topsecret", "permissions": "op"},
        "john": {"password": "secret", "permissions": "present"},
        "james": {"password": "secret2", "permissions": "observe"},
        "peter": {"password": "secret4"}
    },
    "fallback-users": [
        {"permissions": "observe", "password": {"type":"wildcard"}}
    ]
}`

func TestDescriptionJSON(t *testing.T) {
	var d Description
	err := json.Unmarshal([]byte(descJSON), &d)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	dd, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var ddd Description
	err = json.Unmarshal([]byte(dd), &ddd)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(d, ddd) {
		t.Errorf("Got %v, expected %v", ddd, d)
	}
}

var obsoleteJSON = `
{
   "op": [{"username": "jch","password": "topsecret"}],
   "max-history-age": 10,
   "allow-subgroups": true,
   "presenter": [
       {"username": "john", "password": "secret"}
   ],
   "other": [
       {"username": "james", "password": "secret2"},
       {"username": "peter", "password": "secret4"},
       {}
   ]
}`

func TestUpgradeDescription(t *testing.T) {
	var d1 Description
	err := json.Unmarshal([]byte(descJSON), &d1)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var d2 Description
	err = json.Unmarshal([]byte(obsoleteJSON), &d2)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	err = upgradeDescription(&d2)
	if err != nil {
		t.Fatalf("upgradeDescription: %v", err)
	}

	if d2.Op != nil || d2.Presenter != nil || d2.Other != nil {
		t.Errorf("legacy field is not nil")
	}

	if len(d1.Users) != len(d2.Users) {
		t.Errorf("length not equal: %v != %v",
			len(d1.Users), len(d2.Users))
	}

	for k, v1 := range d1.Users {
		v2 := d2.Users[k]
		if !reflect.DeepEqual(v1.Password, v2.Password) ||
			!permissionsEqual(
				v1.Permissions.Permissions(&d1),
				v2.Permissions.Permissions(&d2),
			) {
			t.Errorf("%v not equal: %v != %v", k, v1, v2)
		}
	}
}

var jch = "jch"
var john = "john"
var james = "james"
var paul = "paul"
var peter = "peter"

var badClients = []ClientCredentials{
	{Username: &jch, Password: "foo"},
	{Username: &john, Password: "foo"},
	{Username: &james, Password: "foo"},
}

type credPerm struct {
	c ClientCredentials
	p []string
}

var goodClients = []credPerm{
	{
		ClientCredentials{Username: &jch, Password: "topsecret"},
		[]string{"op", "present", "token"},
	},
	{
		ClientCredentials{Username: &john, Password: "secret"},
		[]string{"present"},
	},
	{
		ClientCredentials{Username: &james, Password: "secret2"},
		[]string{},
	},
	{
		ClientCredentials{Username: &paul, Password: "secret3"},
		[]string{},
	},
	{
		ClientCredentials{Username: &peter, Password: "secret4"},
		[]string{},
	},
}

func TestPermissions(t *testing.T) {
	var g Group
	err := json.Unmarshal([]byte(descJSON), &g.description)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, c := range badClients {
		t.Run("bad "+*c.Username, func(t *testing.T) {
			_, p, err := g.GetPermission(c)
			if err != ErrNotAuthorised {
				t.Errorf("GetPermission %v: %v %v", c, err, p)
			}
		})
	}

	for _, cp := range goodClients {
		t.Run("good "+*cp.c.Username, func(t *testing.T) {
			u, p, err := g.GetPermission(cp.c)
			if err != nil {
				t.Errorf("GetPermission %v: %v", cp.c, err)
			} else if u != *cp.c.Username ||
				!permissionsEqual(p, cp.p) {
				t.Errorf("%v: got %v %v, expected %v",
					cp.c, u, p, cp.p)
			}
		})
	}

}

func TestExtraPermissions(t *testing.T) {
	j := `
{
    "users": {
        "jch": {"password": "topsecret", "permissions": "op"},
        "john": {"password": "secret", "permissions": "present"},
        "james": {"password": "secret2", "permissions": "observe"}
    }
}`

	var d Description
	err := json.Unmarshal([]byte(j), &d)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	doit := func(u string, p []string) {
		pu := d.Users[u].Permissions.Permissions(&d)
		if !permissionsEqual(pu, p) {
			t.Errorf("%v: expected %v, got %v", u, p, pu)
		}
	}

	doit("jch", []string{"op", "token", "present"})
	doit("john", []string{"present"})
	doit("james", []string{})

	d.AllowRecording = true
	d.UnrestrictedTokens = false

	doit("jch", []string{"op", "record", "token", "present"})
	doit("john", []string{"present"})
	doit("james", []string{})

	d.AllowRecording = false
	d.UnrestrictedTokens = true

	doit("jch", []string{"op", "token", "present"})
	doit("john", []string{"token", "present"})
	doit("james", []string{})

	d.AllowRecording = true
	d.UnrestrictedTokens = true

	doit("jch", []string{"op", "record", "token", "present"})
	doit("john", []string{"token", "present"})
	doit("james", []string{})
}

func TestUsernameTaken(t *testing.T) {
	var g Group
	err := json.Unmarshal([]byte(descJSON), &g.description)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if g.UserExists("") {
		t.Error("UserExists(\"\") is true, expected false")
	}
	if !g.UserExists("john") {
		t.Error("UserExists(john) is false")
	}
	if !g.UserExists("john") {
		t.Error("UserExists(james) is false")
	}
	if g.UserExists("paul") {
		t.Error("UserExists(paul) is true")
	}
}

func TestFmtpValue(t *testing.T) {
	type fmtpTest struct {
		fmtp  string
		key   string
		value string
	}
	fmtpTests := []fmtpTest{
		{"", "foo", ""},
		{"profile-id=0", "profile-id", "0"},
		{"profile-id=0", "foo", ""},
		{"level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f", "profile-level-id", "42001f"},
		{"foo=1;bar=2;quux=3", "foo", "1"},
		{"foo=1;bar=2;quux=3", "bar", "2"},
		{"foo=1;bar=2;quux=3", "fu", ""},
	}

	for _, test := range fmtpTests {
		v := fmtpValue(test.fmtp, test.key)
		if v != test.value {
			t.Errorf("fmtpValue(%v, %v) = %v, expected %v",
				test.fmtp, test.key, v, test.value,
			)
		}
	}
}

func TestValidGroupName(t *testing.T) {
	type nameTest struct {
		name   string
		result bool
	}
	tests := []nameTest{
		{"", false},
		{"/", false},
		{"/foo", false},
		{"foo/", false},
		{"./foo", false},
		{"foo/.", false},
		{"../foo", false},
		{"foo/..", false},
		{"foo/./bar", false},
		{"foo/../bar", false},
		{"foo", true},
		{"foo/bar", true},
	}

	for _, test := range tests {
		r := validGroupName(test.name)
		if r != test.result {
			t.Errorf("Valid %v: got %v, expected %v",
				test.name, r, test.result)
		}
	}
}
