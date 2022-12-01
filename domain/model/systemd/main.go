package systemd

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"systemd-cd/domain/model/toml"
)

var (
	ErrNoSuchFileOrDir       = errors.New("no such file or directory")
	ErrUnitFileNotManaged    = errors.New("unit file not managed by systemd-cd")
	ErrUnitEnvFileNotManaged = errors.New("unit env file not managed by systemd-cd")
)

type ISystemd interface {
	NewService(name string, uf UnitFileService, env map[string]string) (UnitService, error)
	DeleteService(u UnitService) error

	loadUnitFileSerivce(path string) (u UnitFileService, isGeneratedBySystemdCd bool, err error)
	writeUnitFileService(u UnitFileService, path string) error

	loadEnvFile(path string) (e map[string]string, isGeneratedBySystemdCd bool, err error)
	writeEnvFile(e map[string]string, path string) error
}

func New(s Systemctl, unitFileDir string) (ISystemd, error) {
	// check `unitFileDir`
	// TODO: if invalid dir path, print warning
	err := MkdirIfNotExist(unitFileDir)
	if err != nil {
		return Systemd{}, err
	}

	if !strings.HasSuffix(unitFileDir, "/") {
		// add trailing slash
		unitFileDir += "/"
	}
	return Systemd{s, unitFileDir}, nil
}

type Systemd struct {
	systemctl   Systemctl
	unitFileDir string
}

// Generate unit-file.
// If unit-file already exists, replace it.
func (s Systemd) NewService(name string, uf UnitFileService, env map[string]string) (UnitService, error) {
	// load unit file
	path := strings.Join([]string{s.unitFileDir, name, ".service"}, "")
	loaded, isGeneratedBySystemdCd, err := s.loadUnitFileSerivce(path)
	if err != nil && !os.IsNotExist(err) {
		// fail
		return UnitService{}, err
	}

	if os.IsNotExist(err) {
		// unit file not exists
		// generate `.service` file to `path`
		err = s.writeUnitFileService(uf, path)
	} else if isGeneratedBySystemdCd {
		// unit file already exists and file generated by systemd-cd
		if !loaded.Equals(uf) {
			// file has changes
			// update `.service` file to `path`
			err = s.writeUnitFileService(uf, path)
		}
	} else {
		// unit file already exists and file not generated by systemd-cd
		err = ErrUnitFileNotManaged
	}
	if err != nil {
		// fail
		return UnitService{}, err
	}

	if uf.Service.EnvironmentFile != nil {
		// load env file
		envPath := *uf.Service.EnvironmentFile
		loaded, isGeneratedBySystemdCd, err := s.loadEnvFile(envPath)
		if err != nil && !os.IsNotExist(err) {
			// fail
			return UnitService{}, err
		}

		if os.IsNotExist(err) {
			// unit file not exists
			// generate env file to `envPath`
			err = s.writeEnvFile(env, envPath)
		} else if isGeneratedBySystemdCd {
			// unit file already exists and file generated by systemd-cd
			if !reflect.DeepEqual(env, loaded) {
				// file has changes
				// update env file to `envPath`
				err = s.writeEnvFile(env, envPath)
			}
		} else {
			// unit file already exists and file not generated by systemd-cd
			err = ErrUnitEnvFileNotManaged
		}
		if err != nil {
			// fail
			return UnitService{}, err
		}
	}

	// daemon-reload
	err = s.systemctl.DaemonReload()

	return UnitService{s.systemctl, name, uf, path, env}, err
}

func (s Systemd) DeleteService(u UnitService) error {
	err := u.Disable(true)
	if err != nil {
		return err
	}

	// Delete `.service` file
	err = os.Remove(u.Path)

	return err
}

func (s Systemd) loadUnitFileSerivce(path string) (u UnitFileService, isGeneratedBySystemdCd bool, err error) {
	// Read file
	b := &bytes.Buffer{}
	err = ReadFile(path, b)
	if err != nil {
		return
	}

	// Check generator
	if strings.Contains(b.String(), "#! Generated by systemd-cd\n") {
		isGeneratedBySystemdCd = true
	}

	// Unmarshal
	u, err = UnmarshalUnitFile(b)

	return
}

func (s Systemd) writeUnitFileService(u UnitFileService, path string) error {
	// Marshal
	b := &bytes.Buffer{}
	// Add annotation to distinct generator
	b.WriteString("#! Generated by systemd-cd\n")
	if b2, err := MarshalUnitFile(u); err != nil {
		return err
	} else {
		b.Write(b2)
	}

	// Write to file
	err := WriteFile(path, b.Bytes())

	return err
}

func (s Systemd) loadEnvFile(path string) (e map[string]string, isGeneratedBySystemdCd bool, err error) {
	// Read file
	b := &bytes.Buffer{}
	err = ReadFile(path, b)
	if err != nil {
		return
	}

	// Check generator
	if strings.Contains(b.String(), "#! Generated by systemd-cd\n") {
		isGeneratedBySystemdCd = true
	}

	// Decode
	err = toml.Decode(b, &e)
	if err != nil {
		return
	}

	return
}

func (s Systemd) writeEnvFile(e map[string]string, path string) error {
	// Encode
	b := &bytes.Buffer{}
	// Add annotation
	b.WriteString("#! Generated by systemd-cd\n")
	indent := ""
	err := toml.Encode(b, e, toml.EncodeOption{Indent: &indent})
	if err != nil {
		return err
	}

	// Write to file
	err = WriteFile(path, b.Bytes())

	return err
}
