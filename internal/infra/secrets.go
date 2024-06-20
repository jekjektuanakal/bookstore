package infra

import "github.com/kelseyhightower/envconfig"

type EnvSecrets struct {
	authKey string
}

func NewEnvSecrets() EnvSecrets {
	secrets := struct {
		AuthKey string `required:"true"`
	}{}
	envconfig.MustProcess("BOOKSTORE_SECRETS", &secrets)

	return EnvSecrets{
		authKey: secrets.AuthKey,
	}
}

func (s *EnvSecrets) GetAuthKey() string {
	return s.authKey
}
