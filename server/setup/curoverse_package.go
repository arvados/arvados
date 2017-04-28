package setup

import (
	"fmt"
	"os"
)

func (s *Setup) installCuroversePackage(name string) error {
	rel := s.Agent.ArvadosAptRepo.Release
	if rel == "" {
		return fmt.Errorf("os release not known: cannot add arvados package repo")
	}
	listFn := "/etc/apt/sources.list.d/arvados.list"
	{
		err := command("apt-key", "adv", "--keyserver", "pool.sks-keyservers.net", "--recv", "1078ECD7").Run()
		if err != nil {
			return err
		}
		_, err = os.Stat(listFn)
		if os.IsNotExist(err) {
			err = atomicWriteFile(listFn, []byte(fmt.Sprintf("deb http://apt.arvados.org/ %s main\n", rel)), 0644)
		}
		if err != nil {
			return err
		}
		err = command("apt-get", "update").Run()
		if err != nil {
			os.Remove(listFn)
			return err
		}
	}
	return (&osPackage{
		Debian: name,
		RedHat: name,
	}).install()
}
