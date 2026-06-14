//go:build unix

package discovery

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"sync"
)

const serviceUserName = "nms-agent"

var (
	serviceUserOnce sync.Once
	serviceUserUID  int
	serviceUserGID  int
	serviceUserErr  error
)

func ChownGeneratedArtifact(path string) error {
	return chownToServiceUser(path)
}

func chownToServiceUser(path string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	serviceUserOnce.Do(func() {
		u, err := user.Lookup(serviceUserName)
		if err != nil {
			serviceUserErr = fmt.Errorf("lookup %s user: %w", serviceUserName, err)
			return
		}
		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			serviceUserErr = fmt.Errorf("parse %s uid %q: %w", serviceUserName, u.Uid, err)
			return
		}
		gid, err := strconv.Atoi(u.Gid)
		if err != nil {
			serviceUserErr = fmt.Errorf("parse %s gid %q: %w", serviceUserName, u.Gid, err)
			return
		}
		serviceUserUID = uid
		serviceUserGID = gid
	})
	if serviceUserErr != nil {
		return serviceUserErr
	}
	return os.Chown(path, serviceUserUID, serviceUserGID)
}
