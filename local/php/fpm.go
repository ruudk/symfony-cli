package php

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/symfony-cli/terminal"
)

func (p *Server) defaultFPMConf() string {
	logLevel := "notice"
	if terminal.IsDebug() {
		logLevel = "debug"
	}
	userConfig := ""
	// when root, we need to configure the user in FPM configuration
	if os.Geteuid() == 0 {
		uid := "nobody"
		gid := "nobody"
		users := []string{
			"www-data", // debian-like, alpine
			"apache",   // fedora
			"http",     // pld linux
			"www",      // freebsd?
			"_www",     // macOS
		}
		// we prefer to use the currently logged-in user (which might not be the current user when using sudo)
		// also, you might not be logged in (when running in a Docker container for instance), in which case, we need to fall back to a "default" user
		cmd := exec.Command("logname")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err == nil {
			users = append([]string{strings.TrimSpace(out.String())}, users...)
		}
		for _, name := range users {
			u, err := user.Lookup(name)
			if err == nil {
				uid = u.Uid
				gid = u.Gid
				break
			}
		}
		userConfig = fmt.Sprintf("user = %s\ngroup = %s", uid, gid)
	}
	minVersion, _ := version.NewVersion("7.3.0")
	workerConfig := ""
	logLimit := ""
	if p.Version.FullVersion.GreaterThanOrEqual(minVersion) {
		workerConfig = "decorate_workers_output = no"
		// see https://github.com/docker-library/php/pull/725#issuecomment-443540114
		logLimit = "log_limit = 8192"
	}
	listen := p.addr
	if listen[0] == ':' {
		listen = "127.0.0.1" + listen
	}
	return fmt.Sprintf(`
[global]
error_log = /dev/fd/2
log_level = %s
daemonize = no
%s

[www]
%s
listen = %s
listen.allowed_clients = 127.0.0.1
pm = dynamic
pm.max_children = 5
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
pm.status_path = /__php-fpm-status__

; Ensure worker stdout and stderr are sent to the main error log
catch_workers_output = yes
%s

php_admin_value[error_log] = /dev/fd/2
php_admin_flag[log_errors] = on

; we want to expose env vars (like in FOO=bar symfony server:start)
clear_env = no
`, logLevel, logLimit, userConfig, listen, workerConfig)
}

func (p *Server) fpmConfigFile() string {
	path := filepath.Join(p.homeDir, fmt.Sprintf("php/%s/fpm-%s.ini", name(p.projectDir), p.Version.Version))
	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(path), 0755)
	}
	return path
}
