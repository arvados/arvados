package setup

import (
	"os"
	"path"
)

func (s *Setup) generateSelfSignedCert() error {
	path := path.Join(s.DataDir, "certs")
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	for _, f := range []func() error{
		command("openssl", "genrsa", "-out", path+"/selfsigned.key", "2048").Run,
		command("openssl", "req", "-new", "-batch", "-subj", "/C=US/ST=MA/O=Example, Inc./CN=example.com", "-key", path+"/selfsigned.key", "-out", path+"/selfsigned.csr").Run,
		command("openssl", "x509", "-req", "-days", "3650", "-in", path+"/selfsigned.csr", "-signkey", path+"/selfsigned.key", "-out", path+"/selfsigned.crt").Run,
	} {
		err := f()
		if err != nil {
			return err
		}
	}
	return nil
}
