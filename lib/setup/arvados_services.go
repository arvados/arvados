package setup

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"time"

	consul "github.com/hashicorp/consul/api"
)

func (s *Setup) installPassenger() error {
	if _, err := os.Stat("/etc/nginx/conf.d/passenger.conf"); err == nil {
		return nil
	}

	err := command("apt-get", "install", "-y", "apt-transport-https", "ca-certificates").Run()
	if err != nil {
		return err
	}
	err = atomicWriteFile("/etc/apt/sources.list.d/passenger.list", []byte("deb https://oss-binaries.phusionpassenger.com/apt/passenger jessie main\n"), 0644)
	if err != nil {
		return err
	}
	err = command("apt-get", "update").Run()
	if err != nil {
		return err
	}
	err = command("apt-get", "install", "-y", "nginx-extras", "passenger").Run()
	if err != nil {
		return err
	}
	err = os.Symlink("../passenger.conf", "/etc/nginx/conf.d/passenger.conf")
	if err != nil {
		return err
	}
	return command("service", "nginx", "restart").Run()
}

func (s *Setup) installArvadosServices() error {
	var todo []func() error
	var cvPkgs []string
	if s.RunAPI {
		err := (&osPackage{Debian: "curl"}).install()
		if err != nil {
			return err
		}
		resp, err := http.Get("https://get.rvm.io")
		if err != nil {
			return err
		}
		rvmCmd := command("bash", "-s", "stable", "--ruby=2.3")
		rvmCmd.Stdin = resp.Body

		todo = append(todo,
			command("bash", "-c", `gpg --keyserver hkp://keys.gnupg.net --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3`).Run,
			rvmCmd.Run,
		)
		todo = append(todo,
			command("apt-get", "install", "-y", "--no-install-recommends",
				"bison", "build-essential", "libcurl4-openssl-dev", "git").Run,
		)

		err = (&osPackage{Debian: "postgresql"}).install()
		if err != nil {
			return err
		}

		wwwUser, err := user.Lookup("www-data")
		if err != nil {
			return err
		}
		wwwGID, err := strconv.Atoi(wwwUser.Gid)
		if err != nil {
			return err
		}

		gitRepoDir := path.Join(s.Agent.DataDir, "git", "repositories")
		{
			if err := os.MkdirAll(gitRepoDir, 0755); err != nil {
				return err
			}
			err = os.Chown(gitRepoDir, 0, wwwGID)
			if err != nil {
				return err
			}
		}

		appYml := "/etc/arvados/api/application.yml"
		{
			secretToken, err := s.newSecret(32)
			if err != nil {
				return err
			}
			blobSigningKey, err := s.newSecret(32)
			if err != nil {
				return err
			}
			err = atomicWriteJSON(appYml, map[string]interface{}{
				"production": map[string]interface{}{
					"uuid_prefix":          s.Agent.ClusterID,
					"secret_token":         secretToken,
					"blob_signing_key":     blobSigningKey,
					"sso_app_secret":       "TODO",
					"sso_app_id":           "TODO",
					"sso_provider_url":     "https://TODO/",
					"workbench_address":    "https://TODO/",
					"websocket_address":    "wss://TODO/",
					"git_repositories_dir": gitRepoDir,
					"git_internal_dir":     path.Join(s.Agent.DataDir, "internal.git"),
				}}, 0640)
			err = os.Chown(appYml, 0, wwwGID)
			if err != nil {
				return err
			}
		}

		dbYml := "/etc/arvados/api/database.yml"
		if _, err = os.Stat(dbYml); err != nil && !os.IsNotExist(err) {
			return err
		} else if os.IsNotExist(err) {
			saidWaiting := false
		waitPg:
			for {
				err := command("pg_isready").Run()
				switch err := err.(type) {
				case nil:
					break waitPg
				case *exec.ExitError:
					if !saidWaiting {
						err := command("service", "postgresql", "start").Run()
						if err != nil {
							return err
						}
						log.Print("waiting for postgres to be ready")
						saidWaiting = true
					}
					time.Sleep(time.Second)
					continue
				default:
					return err
				}
			}
			password, err := s.newSecret(16)
			if err != nil {
				return err
			}
			// TODO: write password to a file here, so if
			// running "create user" succeeds but writing
			// database.yml fails, we can recover on a
			// subsequent attempt.

			sql := fmt.Sprintf("create user arvados with createdb encrypted password '%s'", password)
			cmd := command("su", "-c", "psql", "postgres")
			cmd.Stdin = bytes.NewBufferString(sql)
			err = cmd.Run()
			if err != nil {
				return err
			}
			err = atomicWriteJSON(dbYml, map[string]interface{}{
				"production": map[string]interface{}{
					"adapter":  "postgresql",
					"template": "template0",
					"encoding": "utf8",
					"database": "arvados_" + s.Agent.ClusterID,
					"username": "arvados",
					"password": password,
					"host":     "localhost",
				}}, 0640)
			if err != nil {
				return err
			}
			err = os.Chown(dbYml, 0, wwwGID)
			if err != nil {
				return err
			}
		}
		cvPkgs = append(cvPkgs,
			"arvados-api-server",
		)
		todo = append(todo,
			command("apt-key", "adv", "--keyserver", "hkp://keyserver.ubuntu.com:80", "--recv-keys", "561F9B9CAC40B2F7").Run,
			s.installPassenger,
			func() error {
				return s.consulTemplateTrigger("arvados-api", "/etc/nginx/sites-available/arvados-api.conf", tmplNginxAPI, 0640, "service nginx reload")
			},
			func() error {
				return os.Symlink("../sites-available/arvados-api.conf", "/etc/nginx/sites-enabled/arvados-api.conf")
			})
		c, err := s.ConsulMaster()
		if err != nil {
			return err
		}
		err = c.Agent().ServiceRegister(&consul.AgentServiceRegistration{
			ID:   "arvados-api:" + s.LANHost,
			Name: "arvados-api",
			Port: s.Agent.Ports.API,
			Checks: consul.AgentServiceChecks{
				{
					HTTP:     fmt.Sprintf("http://127.0.0.1:%d/discovery/v1/apis/arvados/v1/rest", s.Agent.Ports.API),
					Interval: "15s",
				},
			},
		})
		if err != nil {
			return err
		}
	}
	for _, pkg := range cvPkgs {
		pkg := pkg
		todo = append(todo, func() error {
			return s.installCuroversePackage(pkg)
		})
	}
	// f = append(f, func() error {
	// 	return s.consulTemplateExec("arvados-agent", "./agent.json", tmplArvadosAgent, 0640, "arvados-admin", "agent", "-config", "./agent.json")
	// })
	for _, f := range todo {
		err := f()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Setup) newSecret(n int) (string, error) {
	buf := &bytes.Buffer{}
	_, err := io.CopyN(buf, rand.Reader, 16)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf), nil
}
