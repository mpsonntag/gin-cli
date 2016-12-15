package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/G-Node/gin-cli/web"
	"github.com/G-Node/gin-core/gin"
)

func TestMain(m *testing.M) {
	tmpdir, err := ioutil.TempDir("", "gintests")
	if err != nil {
		fmt.Fprint(os.Stderr, "Unable to create temporary directory for test setup.")
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	err = os.Setenv("XDG_CONFIG_HOME", tmpdir)
	if err != nil {
		fmt.Fprint(os.Stderr, "Error setting XDG_CONFIG_HOME environment variable for tests.")
		os.Exit(1)
	}

	res := m.Run()
	_ = os.RemoveAll(tmpdir)
	os.Exit(res)
}

func getAccountHandler(w http.ResponseWriter, r *http.Request) {
	aliceInfo := `{"url":"test_server/api/accounts/alice","uuid":"alice_test_uuid","login":"alice","title":null,"first_name":"Alice","middle_name":null,"last_name":"Goodwill",%s"created_at":"2016-11-10T12:26:04.57208Z","updated_at":"2016-11-10T12:26:04.57208+01:00"}`
	aliceAffil := `"affiliation":{"institute":"The Institute","department":"Some department","city":"Munich","country":"Germany","is_public":true},`
	if r.URL.Path == "/api/accounts/alice" {
		fmt.Fprintf(w, aliceInfo, "")
	} else if r.URL.Path == "/api/accounts/alicewithaffil" {
		fmt.Fprintf(w, aliceInfo, aliceAffil)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestRequestAccount(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(getAccountHandler))
	defer ts.Close()

	authcl := NewClient(ts.URL)

	// alice (no affiliation)
	acc, err := authcl.RequestAccount("alice")

	if err != nil {
		t.Errorf("[Account lookup: alice] Request returned error [%s] when it should have succeeded.", err.Error())
	}

	respOK := acc.Login == "alice" && acc.UUID == "alice_test_uuid" && acc.Title == nil &&
		acc.FirstName == "Alice" && acc.MiddleName == nil && acc.LastName == "Goodwill"

	if !respOK {
		t.Error("[Account lookup: alice] Test failed. Response does not match expected values.")
	}

	// alice (with affiliation)
	acc, err = authcl.RequestAccount("alicewithaffil")

	if err != nil {
		t.Errorf("[Account lookup: alicewithaffil] Request returned error [%s] when it should have succeeded.", err.Error())
	}

	affil := acc.Affiliation
	respOK = affil.Institute == "The Institute" && affil.Department == "Some department" &&
		affil.City == "Munich" && affil.Country == "Germany"

	if !respOK {
		t.Error("[Account lookup: alicewithaffil] Test failed. Response does not match expected values.")
	}

	// non-existent user
	acc, err = authcl.RequestAccount("I don't exist")
	if err == nil {
		t.Error("[Account lookup] Non existent account request succeeded when it should have failed.")
	}

	var emptyAcc gin.Account
	if acc != emptyAcc {
		t.Errorf("[Account lookup] Non existent account request returned non-empty account info. [%+v]", acc)
	}

	// server error
	authcl = NewClient("")
	acc, err = authcl.RequestAccount("server is broken")
	if err == nil {
		t.Error("[Account lookup] Request succeeded when it should have failed.")
	}

	if acc != emptyAcc {
		t.Errorf("[Account lookup] Bad request returned non-empty account info. [%+v]", acc)
	}
}

func getKeysHandler(w http.ResponseWriter, r *http.Request) {
	aliceKeys := `[{"url":"test_server/api/keys?fingerprint=fingerprint_one","fingerprint":"fingerprint_one","key":"ssh-rsa SSHKEY12344567 name@host","description":"name@host","login":"alice","account_url":"test_server/api/accounts/alice","created_at":"2016-12-12T18:11:54.131134+01:00","updated_at":"2016-12-12T18:11:54.131134+01:00"},{"url":"test_server/api/keys?fingerprint=fingerprint_two","fingerprint":"fingerprint_two","key":"ssh-rsa SSHKEYTHESECONDONE name@host","description":"name@host_2","login":"alice","account_url":"test_server/api/accounts/alice","created_at":"2016-12-12T18:11:54.131134+01:00","updated_at":"2016-12-12T18:11:54.131134+01:00"}]`
	if r.URL.Path == "/api/accounts/alice/keys" {
		fmt.Fprint(w, aliceKeys)
	} else if r.URL.Path == "/api/accounts/errorinducer/keys" {
		http.Error(w, "Server returned error", http.StatusInternalServerError)
	} else if r.URL.Path == "/api/accounts/badresponse/keys" {
		fmt.Fprint(w, "not_json_response")
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestRequestKeys(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(getKeysHandler))
	defer ts.Close()

	authcl := NewClient(ts.URL)

	// alice with 2 keys
	aliceToken := web.UserToken{Username: "alice", Token: "some_sort_of_token"}
	err := aliceToken.StoreToken()
	if err != nil {
		t.Error("[Key retrieval] Error storing token for alice.")
	}

	keys, err := authcl.GetUserKeys()
	if err != nil {
		t.Errorf("[Key retrieval] Request returned error [%s] when it should have succeeded.", err.Error())
	}

	nkeys := 2
	if len(keys) != nkeys {
		t.Errorf("[Key retrieval] Expected %d keys. Got %d.", nkeys, len(keys))
		t.FailNow()
	}

	respOK := keys[0].Key == "ssh-rsa SSHKEY12344567 name@host" &&
		keys[0].Description == "name@host" && keys[0].Login == "alice" &&
		keys[0].Fingerprint == "fingerprint_one"
	respOK = respOK && keys[1].Key == "ssh-rsa SSHKEYTHESECONDONE name@host" &&
		keys[1].Description == "name@host_2" && keys[1].Login == "alice" &&
		keys[1].Fingerprint == "fingerprint_two"

	if !respOK {
		t.Error("[Key retrieval] Test failed. Response does not match expected values.")
	}

	// non-existent user
	nexuser := web.UserToken{Username: "I do not exist", Token: "some_sort_of_token"}
	err = nexuser.StoreToken()
	if err != nil {
		t.Error("[Key retrieval] Error storing token for non existent account test.")
	}

	keys, err = authcl.GetUserKeys()
	if err == nil {
		t.Error("[Key retrieval] Non existent account request succeeded when it should have failed.")
	}

	if len(keys) != 0 {
		t.Errorf("[Key retrieval] Non existent user key request returned non-empty key slice. [%d items]", len(keys))
	}

	// error inducing request
	errorUser := web.UserToken{Username: "errorinducer", Token: "some_sort_of_token"}
	err = errorUser.StoreToken()
	if err != nil {
		t.Error("[Key retrieval] Error storing token for error test.")
	}

	keys, err = authcl.GetUserKeys()
	if err == nil {
		t.Error("[Key retrieval] Request succeeded when it should have failed.")
	}

	if len(keys) != 0 {
		t.Errorf("[Key retrieval] Bad request returned non-empty key slice. [%d items]", len(keys))
	}

	// bad response
	badResponseUser := web.UserToken{Username: "badresponse", Token: "some_sort_of_token"}
	err = badResponseUser.StoreToken()
	if err != nil {
		t.Error("[Key retrieval] Error storing token for bad response test.")
	}

	keys, err = authcl.GetUserKeys()
	if err == nil {
		t.Error("[Key retrieval] Request succeeded when it should have failed.")
	}

	if len(keys) != 0 {
		t.Errorf("[Key retrieval] Bad request returned non-empty key slice. [%d items]", len(keys))
	}

	// not logged in
	oldconf := os.Getenv("XDG_CONFIG_HOME")
	err = os.Setenv("XDG_CONFIG_HOME", "")
	if err != nil {
		t.Error("Error setting XDG_CONFIG_HOME to empty string.")
	}
	keys, err = authcl.GetUserKeys()
	if err == nil {
		t.Error("[Key retrieval] Request without login succeeded when it should have failed.")
	}

	if len(keys) != 0 {
		t.Errorf("[Key retrieval] Request without login returned non-empty key slice. [%d items]", len(keys))
	}

	err = os.Setenv("XDG_CONFIG_HOME", oldconf)
	if err != nil {
		t.Errorf("Error resetting XDG_CONFIG_HOME after no login test.")
	}

	// server error
	authcl = NewClient("")
	nullToken := web.UserToken{Username: "", Token: ""}
	err = nullToken.StoreToken()
	if err != nil {
		t.Error("[Key retrieval] Error storing null token.")
	}
	keys, err = authcl.GetUserKeys()
	if err == nil {
		t.Error("[Key retrieval] Request with bad server succeeded when it should have failed.")
	}

	if len(keys) != 0 {
		t.Errorf("[Key retrieval] Request with bad server returned non-empty key slice. [%d items]", len(keys))
	}
}

func addKeyHandler(w http.ResponseWriter, r *http.Request) {
	match, _ := regexp.MatchString("/api/accounts/[a-zA-Z]+/keys", r.URL.Path)
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in request handler for AddKey test")
	}

	newKey := &gin.SSHKey{}
	if match {
		err := json.Unmarshal(b, newKey)
		if err != nil {
			http.Error(w, "Bad data", http.StatusBadRequest)
		}
		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestAddKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(addKeyHandler))
	defer ts.Close()

	authcl := NewClient(ts.URL)
	aliceToken := web.UserToken{Username: "alice", Token: "some_sort_of_token"}
	err := aliceToken.StoreToken()
	if err != nil {
		t.Error("[Key retrieval] Error storing token for alice.")
	}

	err = authcl.AddKey("KEY123", "a test key", false)
	if err != nil {
		t.Errorf("[Add key] Function returned error: %s", err.Error())
	}

	authcl = NewClient("")
	err = authcl.AddKey("", "", false)
	if err == nil {
		t.Error("[Add key] Request with bad server succeeded when it should have failed.")
	}

	oldconf := os.Getenv("XDG_CONFIG_HOME")
	err = os.Setenv("XDG_CONFIG_HOME", "")
	if err != nil {
		t.Error("Error setting XDG_CONFIG_HOME to empty string.")
	}
	err = authcl.AddKey("", "", false)
	if err == nil {
		t.Error("[Add key] Request without login succeeded when it should have failed.")
	}
	err = os.Setenv("XDG_CONFIG_HOME", oldconf)
	if err != nil {
		t.Errorf("Error resetting XDG_CONFIG_HOME after no login test.")
	}
}
