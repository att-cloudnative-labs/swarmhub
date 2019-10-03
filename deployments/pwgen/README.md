# PWGEN
This is for generating passwords which will be put into the `localusers.csv` file. Swarmhub will first look inside localusers.csv before reaching out to LDAP.

To create local accounts:
```bash
$ ./pwgen --username=readonly --role=1 --password="my password"
cmVhZG9ubHk=,1,JDJhJDEwJElYVmZsaUhUUXNLdkd2SDRCL29BYy5lVVBSRFFJL0dpU3IvMUdaZHFEYTl1Zkl6bk1PL1VX
$ ./pwgen --username=poweruser --role=5 --password="my password"
cG93ZXJ1c2Vy,5,JDJhJDEwJGExUUhlTGIubW1wZ21BOHhLUnFpeE9sM3VnenY1R3pGN2YvMmdzd0I0WG1Qald5R0ZyeEM2
```

After genereting the data needed for local user accounts add the data to localusers.csv and upload the file to kubernetes as a secret.

Example localusers.csv file
```csv
username,role,password
cmVhZG9ubHk=,1,JDJhJDEwJElYVmZsaUhUUXNLdkd2SDRCL29BYy5lVVBSRFFJL0dpU3IvMUdaZHFEYTl1Zkl6bk1PL1VX
cG93ZXJ1c2Vy,5,JDJhJDEwJGExUUhlTGIubW1wZ21BOHhLUnFpeE9sM3VnenY1R3pGN2YvMmdzd0I0WG1Qald5R0ZyeEM2
```
