package main

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/api"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/storage"
)

// Registry is a local version of viper
var Registry *viper.Viper

// ConfigSet sets the configs based on the environment
func ConfigSet() {
	Registry = viper.New()
	Registry.AutomaticEnv()
	Registry.SetConfigName("settings")
	Registry.AddConfigPath(".")
	if err := Registry.MergeInConfig(); err != nil {
		err = fmt.Errorf("failed to MergeInConfig: %v", err)
		panic(err)
	}
	ldapSet()
	storageSet()
	grafanaSet()

	tlsCertFileLoc = Registry.GetString("TLS_CERT_FILE_LOC")
	tlsKeyFileLoc = Registry.GetString("TLS_KEY_FILE_LOC")
	UserFileLocation = Registry.GetString("USER_FILE_LOCATION")
	htmlDir = Registry.GetString("HTML_DIR")
	db.SourceName = Registry.GetString("DB_SOURCE_NAME")
	db.Set()

	lmSecGroups := Registry.GetStringSlice("LOCUST_MASTER_SECURITY_GROUPS")
	lsSecGroups := Registry.GetStringSlice("LOCUST_SLAVE_SECURITY_GROUPS")
	api.LocustMasterSecurityGroups = convertSliceToBashString(lmSecGroups)
	api.LocustSlaveSecurityGroups = convertSliceToBashString(lsSecGroups)

}

func ldapSet() {
	ldapAccountsEnabled = Registry.GetBool("LDAP_ACCOUNTS_ENABLED")
	localAccountsEnabled = Registry.GetBool("LOCAL_ACCOUNTS_ENABLED")
	if localAccountsEnabled {
		err := loadLoginData(Registry.GetString("LOGIN_DATA_LOCATION"))
		if err != nil {
			log.Fatal("Failed to load login data to map:", err)
		}
	}

	LdapAddress = Registry.GetString("LDAP_ADDRESS")
	ldapInsecureSkip = Registry.GetBool("LDAP_INSECURE_SKIP")
	ldapUseLocalPowerUsers = Registry.GetBool("LDAP_USE_LOCAL_POWER_USERS")
	ldapBind = Registry.GetString("LDAP_BIND")
	ldapBaseDN = Registry.GetString("LDAP_BASE_DN")
	ldapGroupFilter = Registry.GetString("LDAP_GROUP_FILTER")
	ldapAllUsersReadOnly = Registry.GetBool("LDAP_ALL_USERS_READ_ONLY")

	groups := Registry.GetStringSlice("LDAP_POWER_USERS_GROUPS")
	for _, group := range groups {
		ldapPowerUsersGroups[group] = true
	}
}

func storageSet() {
	accessKey := Registry.GetString("AWS_S3_ACCESS_KEY")
	secretAccessKey := Registry.GetString("AWS_S3_SECRET_ACCESS_KEY")
	region := Registry.GetString("AWS_S3_REGION")
	bucket := Registry.GetString("AWS_S3_BUCKET")

	err := storage.SetS3(accessKey, secretAccessKey, region, bucket)
	if err != nil {
		log.Fatal("Failed to performa storage.SetS3:", err)
	}

	storage.ServerSideEncryption = Registry.GetString("AWS_S3_SERVER_SIDE_ENCRYPTION")
}

func grafanaSet() {
	api.GrafanaEnabled = Registry.GetBool("GRAFANA_ENABLED")
	api.GrafanaDomain = Registry.GetString("GRAFANA_DOMAIN")
	api.GrafanaAPIKey = Registry.GetString("GRAFANA_API_KEY")
	api.GrafanaDashboardUID = Registry.GetString("GRAFANA_DASHBOARD_UID")
}

func convertSliceToBashString(list []string) string {
	listBytes := []byte("[")
	for i, v := range list {
	     if i == 0 {
	         listBytes = append(listBytes, "\"" + v + "\""...)
	         continue
	     }
	     listBytes = append(listBytes, ", \"" + v + "\""...)
	}
	listBytes = append(listBytes, "]"...)
	return string(listBytes)
}
