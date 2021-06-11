module git.arvados.org/arvados.git

go 1.13

require (
	github.com/AdRoll/goamz v0.0.0-20170825154802-2731d20f46f4
	github.com/Azure/azure-sdk-for-go v45.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.3
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.1
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.0 // indirect
	github.com/Microsoft/go-winio v0.4.5 // indirect
	github.com/alcortesm/tgz v0.0.0-20161220082320-9c5fe88206d7 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/arvados/cgofuse v1.2.0-arvados1
	github.com/aws/aws-sdk-go v1.25.30
	github.com/aws/aws-sdk-go-v2 v0.23.0
	github.com/bgentry/speakeasy v0.1.0 // indirect
	github.com/bradleypeabody/godap v0.0.0-20170216002349-c249933bc092
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/coreos/go-systemd v0.0.0-20180108085132-cc4f39464dc7
	github.com/creack/pty v1.1.7
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/docker/distribution v2.6.0-rc.1.0.20180105232752-277ed486c948+incompatible // indirect
	github.com/docker/docker v1.4.2-0.20180109013817-94b8a116fbf1
	github.com/docker/go-connections v0.3.0 // indirect
	github.com/docker/go-units v0.3.3-0.20171221200356-d59758554a3d // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/gliderlabs/ssh v0.2.2 // indirect
	github.com/go-asn1-ber/asn1-ber v1.4.1 // indirect
	github.com/go-ldap/ldap v3.0.3+incompatible
	github.com/gogo/protobuf v1.1.1
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.1-0.20180107155708-5bbbb5b2b572
	github.com/hashicorp/golang-lru v0.5.1
	github.com/imdario/mergo v0.3.8-0.20190415133143-5ef87b449ca7
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmcvetta/randutil v0.0.0-20150817122601-2bb1b664bcff
	github.com/jmoiron/sqlx v1.2.0
	github.com/johannesboyne/gofakes3 v0.0.0-20200716060623-6b2b4cb092cc
	github.com/julienschmidt/httprouter v1.2.0
	github.com/kevinburke/ssh_config v0.0.0-20171013211458-802051befeb5 // indirect
	github.com/lib/pq v1.3.0
	github.com/msteinert/pam v0.0.0-20190215180659-f29b9f28d6f9
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1-0.20171125024018-577479e4dc27 // indirect
	github.com/pelletier/go-buffruneio v0.2.0 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/prometheus/client_golang v1.2.1
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0
	github.com/satori/go.uuid v1.2.1-0.20180103174451-36e9d2ebbde5 // indirect
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/src-d/gcfg v1.3.0 // indirect
	github.com/xanzy/ssh-agent v0.1.0 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys v0.0.0-20210603125802-9665404d3644
	golang.org/x/tools v0.1.0 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007
	golang.org/x/tools v0.1.2 // indirect
	google.golang.org/api v0.13.0
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/check.v1 v1.0.0-20161208181325-20d25e280405
	gopkg.in/square/go-jose.v2 v2.3.1
	gopkg.in/src-d/go-billy.v4 v4.0.1
	gopkg.in/src-d/go-git-fixtures.v3 v3.5.0 // indirect
	gopkg.in/src-d/go-git.v4 v4.0.0
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.2.4 // indirect
	rsc.io/getopt v0.0.0-20170811000552-20be20937449
)

replace github.com/AdRoll/goamz => github.com/arvados/goamz v0.0.0-20190905141525-1bba09f407ef

replace gopkg.in/yaml.v2 => github.com/arvados/yaml v0.0.0-20210427145106-92a1cab0904b
