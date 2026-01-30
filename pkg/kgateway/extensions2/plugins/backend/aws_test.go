package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/ir"
)

func TestConfigureAWSAuth_IRSACredentialRefresh(t *testing.T) {
	t.Run("configures AssumeRoleWithWebIdentityProvider when no secret provided", func(t *testing.T) {
		// This simulates IRSA scenario where no explicit secret is provided
		result, err := configureAWSAuth(nil, "us-west-2")
		require.NoError(t, err)
		
		assert.Equal(t, "lambda", result.ServiceName)
		assert.Equal(t, "us-west-2", result.Region)
		
		// Verify AssumeRoleWithWebIdentityProvider is configured for token refresh
		require.NotNil(t, result.CredentialProvider)
		assert.NotNil(t, result.CredentialProvider.AssumeRoleWithWebIdentityProvider)
	})

	t.Run("uses inline credentials when secret provided", func(t *testing.T) {
		secret := &ir.Secret{
			Data: map[string][]byte{
				"accessKey": []byte("test-access-key"),
				"secretKey": []byte("test-secret-key"),
			},
		}
		
		result, err := configureAWSAuth(secret, "us-east-1")
		require.NoError(t, err)
		
		assert.Equal(t, "lambda", result.ServiceName)
		assert.Equal(t, "us-east-1", result.Region)
		
		// Verify inline credentials are used instead of AssumeRoleWithWebIdentity
		require.NotNil(t, result.CredentialProvider)
		assert.NotNil(t, result.CredentialProvider.InlineCredential)
		assert.Nil(t, result.CredentialProvider.AssumeRoleWithWebIdentityProvider)
	})
}