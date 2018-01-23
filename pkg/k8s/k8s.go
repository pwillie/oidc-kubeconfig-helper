package k8s

import (
	"fmt"

	"github.com/ghodss/yaml"
)

type (
	Kubeconfig struct {
		AuthInfos []NamedAuthInfo `json:"users"`
	}

	NamedAuthInfo struct {
		Name     string    `json:"name"`
		AuthInfo *AuthInfo `json:"user"`
	}

	AuthInfo struct {
		AuthProvider *AuthProvider `json:"auth-provider"`
	}

	AuthProvider struct {
		Name               string              `json:"name"`
		AuthProviderConfig *AuthProviderConfig `json:"config"`
	}

	AuthProviderConfig struct {
		IdpIssuerURL string `json:"idp-issuer-url"`
		ClientID     string `json:"client-id"`
		ClientSecret string `json:"client-secret"`
		IDToken      string `json:"id-token"`
		RefreshToken string `json:"refresh-token"`
	}
)

func GenerateUserKubeconfig(email string, issuer string, clientID string, clientSecret string, idToken string, refreshToken string) ([]byte, error) {
	kubeconfig := &Kubeconfig{
		AuthInfos: []NamedAuthInfo{
			NamedAuthInfo{
				Name: email,
				AuthInfo: &AuthInfo{
					AuthProvider: &AuthProvider{
						Name: "oidc",
						AuthProviderConfig: &AuthProviderConfig{
							IdpIssuerURL: issuer,
							ClientID:     clientID,
							ClientSecret: clientSecret,
							IDToken:      idToken,
							RefreshToken: refreshToken,
						},
					},
				},
			},
		},
	}

	output, err := yaml.Marshal(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling yaml: %s", err)
	}
	return output, nil
}
