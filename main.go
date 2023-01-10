package main

import (
	"fmt"
	"os"
	"systemd-cd/application/systemd"
	"systemd-cd/domain/model/logger"
	"systemd-cd/domain/model/logrus"
	"systemd-cd/infrastructure/externalapi/systemctl"
	"time"

	logruss "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

// flags
var (
	logLevel = pflag.String("log.level", "info", "Only log messages with the given severity or above. One of: [panic, fatal, error, warn, info, debug, trace]")
	// varDir                    = pflag.String("storage.var-dir", "/var/lib/systemd-cd/", "Path to variable files")
	// srcDestDir                = pflag.String("storage.src-dir", "/usr/local/systemd-cd/src/", "Path to service source files")
	// binaryDestDir             = pflag.String("storage.binary-dir", "/usr/local/systemd-cd/bin/", "Path to service binary files")
	// etcDestDir                = pflag.String("storage.etc-dir", "/usr/local/systemd-cd/etc/", "Path to service etc files")
	// optDestDir                = pflag.String("storage.opt-dir", "/usr/local/systemd-cd/opt/", "Path to service opt files")
	systemdUnitFileDestDir    = pflag.String("systemd.unit-file-dir", "/usr/local/lib/systemd/system/", "Path to systemd unit files.")
	systemdUnitEnvFileDestDir = pflag.String("systemd.unit-env-file-dir", "/usr/local/systemd-cd/etc/default/", "Path to systemd env files")
	// backupDestDir             = pflag.String("storage.backup-dir", "/var/backups/systemd-cd/", "Path to service backup files")
)

func convertLogLevel(str string) (ok bool, lv logger.Level) {
	switch str {
	case "panic":
		return true, logger.PanicLevel
	case "fatal":
		return true, logger.FatalLevel
	case "error":
		return true, logger.ErrorLevel
	case "warn":
		return true, logger.WarnLevel
	case "info":
		return true, logger.InfoLevel
	case "debug":
		return true, logger.DebugLevel
	case "trace":
		return true, logger.TraceLevel
	default:
		return false, logger.InfoLevel
	}
}

func main() {
	logger.Init(logrus.New(logrus.Param{
		RepeatCaller: func() *bool { var b = true; return &b }(),
		Formatter: &logruss.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		},
	}))

	// parse flags
	pflag.Parse()

	// `--log.level`
	ok, lv := convertLogLevel(*logLevel)
	if !ok {
		logger.Logger().Fatalf("`--log.level` must be specified as \"panic\", \"fatal\", \"error\", \"warn\", \"info\", \"debug\" or \"trace\"")
		os.Exit(1)
	}
	logger.Logger().SetLevel(lv)

	i, err := systemd.New(systemctl.New(), *systemdUnitFileDestDir)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}
	envFile := *systemdUnitEnvFileDestDir + "system-cd-go"
	us, err := i.NewService(
		"systemd-cd-go",
		systemd.UnitFileService{
			Unit: systemd.UnitDirective{
				Description:   "gitops agent for systemd-based linux",
				Documentation: "https://github.com/tingtt/systmed-cd",
				After:         []string{"syslog.target", "network.target"},
				Requires:      nil,
				Wants:         nil,
				Conflicts:     []string{"sendmail.servic", "exim.service"},
			},
			Service: systemd.ServiceDirective{
				Type:            &systemd.UnitTypeSimple,
				EnvironmentFile: &envFile,
				ExecStart:       "watch tail /var/log/syslog",
				ExecStop:        nil,
				ExecReload:      nil,
				Restart:         nil,
				RemainAfterExit: nil,
			},
			Install: systemd.InstallDirective{
				Alias:           nil,
				RequiredBy:      nil,
				WantedBy:        []string{"multi-user.target"},
				Also:            nil,
				DefaultInstance: nil,
			},
		},
		map[string]string{},
	)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}
	s, err := us.GetStatus()
	if err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("status: %v\n", s)
}
