package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Specification struct {
	Stage         string `default:"development"`
	Debug         bool   `default:"true"`
	ServerPort    int    `default:"8080"`
	DbUsername    string `required:"true"`
	DbPassword    string `required:"true"`
	DbHost        string `required:"true"`
	DbPort        int    `required:"true"`
	DbDefaultName string `required:"true"`
	JwtSecret     string `required:"true"`
}

func Load() *Specification {
	var s Specification
	envconfig.MustProcess("APP_GODUTCH", &s)
	return &s
}

func (s *Specification) GetHmacSecret() []byte {
	return []byte(s.JwtSecret)
}

func (s *Specification) GetJwtSecret() string {
	return s.JwtSecret
}

func (s *Specification) GetDbUsername() string {
	return s.DbUsername
}

func (s *Specification) GetDbPassword() string {
	return s.DbPassword
}

func (s *Specification) GetDbHost() string {
	return s.DbHost
}

func (s *Specification) GetDbPort() int {
	return s.DbPort
}

func (s *Specification) GetDbDefaultName() string {
	return s.DbDefaultName
}
