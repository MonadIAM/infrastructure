package seaweedfs

import (
	"errors"
	"os"
	"os/user"
	"strconv"
	"syscall"

	"github.com/MonadIAM/infrastructure/cli/config"
)

func Start(environment config.Environment, args []string) error {
	if len(args) == 0 {
		return errors.New("SeaweedFS server arguments are required after --")
	}

	settings, err := Load(environment)
	if err != nil {
		return err
	}

	err = WriteCredentials(settings, "/tmp/credentials.json")
	if err != nil {
		return err
	}

	err = setCredentialOwner("/tmp/credentials.json")
	if err == nil {
		return syscall.Exec("/entrypoint.sh", append([]string{"/entrypoint.sh"}, args...), os.Environ())
	} else {
		return err
	}
}

func setCredentialOwner(path string) error {
	if os.Geteuid() != 0 {
		return nil
	}

	account, err := user.Lookup("seaweed")
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(account.Gid)
	if err == nil {
		return os.Chown(path, uid, gid)
	} else {
		return err
	}
}
