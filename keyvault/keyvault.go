package keyvault

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
)

// KVSource is ...
type KVSource struct {
	vaultName string
	client    keyvault.BaseClient
}

// New ...
func New(vaultName string) *KVSource {
	authorizer, err := kvauth.NewAuthorizerFromEnvironment()
	if err != nil {
		fmt.Printf("unable to create vault authorizer: %v\n", err)
		os.Exit(1)
	}
	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	return &KVSource{
		vaultName: vaultName,
		client:    basicClient,
	}
}

// GetSecret ...
func (kv KVSource) GetSecret(secname string) string {
	secretResp, err := kv.client.GetSecret(context.Background(), "https://"+kv.vaultName+".vault.azure.net", secname, "")
	if err != nil {
		fmt.Printf("unable to get value for secret: %v\n", err)
		os.Exit(1)
	}

	return *secretResp.Value
}

// GetSecrets ...
func (kv KVSource) GetSecrets() []string {
	secretList, err := kv.client.GetSecrets(context.Background(), "https://"+kv.vaultName+".vault.azure.net", nil)
	if err != nil {
		fmt.Printf("unable to get list of secrets: %v\n", err)
		os.Exit(1)
	}

	result := make([]string, 0)
	for _, secret := range secretList.Values() {
		result = append(result, path.Base(*secret.ID))
	}
	return result
}
