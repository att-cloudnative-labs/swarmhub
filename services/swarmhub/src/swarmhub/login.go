package main

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/jwt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	ldap "gopkg.in/ldap.v2"
)

var (
	localAccountsEnabled     bool
	ldapAccountsEnabled      bool
	ldapUseLocalPowerUsers   bool
	ldapBind                 string
	ldapBaseDN               string
	ldapPowerUsersGroups     = make(map[string]bool)
	ldapReadOnlyGroups       = make(map[string]bool)
	ldapControl              []string
	ldapInsecureSkip         bool
	ldapAllUsersReadOnly     bool
	ldapGroupFilter          string
	localLoginMap            = make(map[string]loginInfo)
	defaultLoginDataLocation = "localusers.csv"

	ErrInvalidUserPass = fmt.Errorf("invalid user/pass")
)

const (
	RolePowerUser = 5
	RoleReadOnly  = 1
	RoleNone      = 0
)

type loginInfo struct {
	Username     string
	PasswordHash []byte
	Role         int
}

func loadLoginData(location string) error {
	if location == "" {
		location = defaultLoginDataLocation
	}
	csvFile, err := os.Open(location)
	if err != nil {
		err = fmt.Errorf("failed to open login data file: %w", err)
		return err
	}
	defer csvFile.Close()

	reader := csv.NewReader(bufio.NewReader(csvFile))
	i := -1
	for {
		record, err := reader.Read()
		i++
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if i == 0 {
			continue
		}
		if len(record) < 3 {
			log.Fatal("Not enough info provided in account record:", record)
		}
		account, err := extractAccount(record[0], record[1], record[2])
		if err != nil {
			log.Fatal("Corrupt account record:", record, err)
		}
		localLoginMap[account.Username] = account
	}
	fmt.Println("local user login data loaded.")
	return nil
}

func extractAccount(userB64, roleStr, passHashB64 string) (loginInfo, error) {
	usernameBytes, err := base64.StdEncoding.DecodeString(userB64)
	if err != nil {
		err = fmt.Errorf("failed to b64 decode username: %w", err)
		return loginInfo{}, err
	}

	passHash, err := base64.StdEncoding.DecodeString(passHashB64)
	if err != nil {
		err = fmt.Errorf("failed to b64 decode password hash: %w", err)
		return loginInfo{}, err
	}

	role, err := strconv.Atoi(roleStr)
	if err != nil {
		err = fmt.Errorf("role string '%v' is not a valid integer: %w", roleStr, err)
		return loginInfo{}, err
	}

	username := string(usernameBytes)
	if username == "" || string(passHash) == "" || role == 0 {
		err = fmt.Errorf("invalid account because one of the fields has a zero value")
		return loginInfo{}, err
	}

	return loginInfo{Username: username, PasswordHash: passHash, Role: role}, nil
}

func login(username string, password string) (signedString string, err error) {
	var role int
	if localAccountsEnabled {
		fmt.Println("Using local accounts")
		role = localLogin(username, password)
	}

	if role > 0 {
		signedString, err = jwt.CreateToken(username, role)
	}

	if ldapAccountsEnabled == true && role == 0 {
		signedString, err = ldapLogin(username, password)
	}

	if signedString == "" && err == nil {
		err = fmt.Errorf("invalid login")
	}
	return signedString, err
}

func localLogin(username, password string) int {
	account, ok := localLoginMap[username]
	if !ok {
		fmt.Println("Failed to find user in localLoginMap")
		return 0
	}

	err := bcrypt.CompareHashAndPassword(account.PasswordHash, []byte(password))
	if err != nil {
		fmt.Println("Incorrect Password.")
		return 0
	}

	return account.Role
}

func ldapLogin(username string, password string) (string, error) {
	role, err := determineRoleFromLDAP(username, password)
	if err != nil {
		return "", err
	}
	if role == RoleNone {
		return "", ErrInvalidUserPass
	}
	signedString, err := jwt.CreateToken(username, role)
	return signedString, err
}

func determineRoleFromLDAP(username string, password string) (int, error) {
	l, err := ldap.DialTLS("tcp", LdapAddress, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		fmt.Println("Dial failed.")
		log.Println(err)
		return RoleNone, err
	}
	defer l.Close()

	err = l.Bind(fmt.Sprintf(ldapBind, username), password)
	if err != nil {
		fmt.Println("username password fail.")
		log.Println(err)
		return RoleNone, err
	}

	if ldapUseLocalPowerUsers == true {
		isPowerUser, _ := checkIfLocalPowerUser(username)
		if isPowerUser {
			return RolePowerUser, nil
		}
	}

	return ldapSearchUserRole(l)
}

func checkIfLocalPowerUser(username string) (bool, error) {
	results, err := usersFromFile()
	if err != nil {
		fmt.Println("Failed the search")
		log.Println(err)
	}

	if results[username] > 0 {
		fmt.Println("Username: " + username + " authorized poweruser.")
		return true, err
	}

	fmt.Println("Username: " + username + " not authorized poweruser.")

	return false, err

}

func usersFromFile() (map[string]int, error) {
	results := make(map[string]int)

	content, err := ioutil.ReadFile(UserFileLocation)
	if err != nil {
		return results, err
	}
	lines := strings.Split(string(content), "\n")

	for _, entry := range lines {
		if len(entry) > 4 {
			results[entry] = 1
		}
	}
	return results, err
}

func ldapSearchUserRole(l *ldap.Conn) (int, error) {
	var defaultRole int
	if ldapAllUsersReadOnly == true {
		defaultRole = RoleReadOnly
	}
	searchRequest := ldap.NewSearchRequest(
		ldapBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		ldapGroupFilter,
		ldapControl,
		nil)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return defaultRole, err
	}

	groups := []string{}
	for _, entry := range sr.Entries {
		for _, attributes := range entry.Attributes {
			for _, attribute := range attributes.Values {
				groups = append(groups, attribute)
			}
		}
	}

	for _, group := range groups {
		if ldapPowerUsersGroups[group] {
			return RolePowerUser, nil
		}
		if ldapReadOnlyGroups[group] {
			return RoleReadOnly, nil
		}
	}

	return defaultRole, nil
}
