package db

import (
	"context"
	"database/sql"

	"github.com/fr0stylo/ddash/internal/db/queries"
)

// GetOrganizationByAuthToken fetches an org by auth token.
func (c *Database) GetOrganizationByAuthToken(ctx context.Context, authToken string) (queries.Organization, error) {
	return c.Queries.GetOrganizationByAuthToken(ctx, authToken)
}

// GetServiceByName fetches a service by name.
func (c *Database) GetServiceByName(ctx context.Context, name string) (queries.Service, error) {
	return c.Queries.GetServiceByName(ctx, name)
}

// GetDefaultOrganization returns the first organization.
func (c *Database) GetDefaultOrganization(ctx context.Context) (queries.Organization, error) {
	return c.Queries.GetDefaultOrganization(ctx)
}

// CreateOrganization inserts a new organization.
func (c *Database) CreateOrganization(ctx context.Context, params queries.CreateOrganizationParams) (queries.Organization, error) {
	return c.Queries.CreateOrganization(ctx, params)
}

// CreateService inserts a new service row.
func (c *Database) CreateService(ctx context.Context, name string) (queries.Service, error) {
	return c.Queries.CreateService(ctx, name)
}

// GetEnvironmentByName fetches an environment by name.
func (c *Database) GetEnvironmentByName(ctx context.Context, name string) (queries.Environment, error) {
	return c.Queries.GetEnvironmentByName(ctx, name)
}

// CreateEnvironment inserts a new environment row.
func (c *Database) CreateEnvironment(ctx context.Context, name string) (queries.Environment, error) {
	return c.Queries.CreateEnvironment(ctx, name)
}

// UpsertServiceInstance upserts a service instance row.
func (c *Database) UpsertServiceInstance(ctx context.Context, params queries.UpsertServiceInstanceParams) (queries.ServiceInstance, error) {
	return c.Queries.UpsertServiceInstance(ctx, params)
}

// CreateDeploymentSimple inserts a deployment with minimal fields.
func (c *Database) CreateDeploymentSimple(ctx context.Context, params queries.CreateDeploymentSimpleParams) (queries.Deployment, error) {
	return c.Queries.CreateDeploymentSimple(ctx, params)
}

// ListServiceInstances returns all service instances.
func (c *Database) ListServiceInstances(ctx context.Context) ([]queries.ListServiceInstancesRow, error) {
	return c.Queries.ListServiceInstances(ctx)
}

// ListServiceInstancesByEnv returns instances in an environment.
func (c *Database) ListServiceInstancesByEnv(ctx context.Context, env string) ([]queries.ListServiceInstancesByEnvRow, error) {
	return c.Queries.ListServiceInstancesByEnv(ctx, env)
}

// ListDeploymentsParams configures the deployments query.
type ListDeploymentsParams = queries.ListDeploymentsParams

// ListPendingCommitsNotInProdParams configures the pending commits query.
type ListPendingCommitsNotInProdParams = queries.ListPendingCommitsNotInProdParams

// ListDeploymentHistoryByServiceParams configures the deployment history query.
type ListDeploymentHistoryByServiceParams = queries.ListDeploymentHistoryByServiceParams

// ListDeployments returns deployment rows.
func (c *Database) ListDeployments(ctx context.Context, params queries.ListDeploymentsParams) ([]queries.ListDeploymentsRow, error) {
	return c.Queries.ListDeployments(ctx, params)
}

// ListServiceFields returns custom fields for a service.
func (c *Database) ListServiceFields(ctx context.Context, serviceID int64) ([]queries.ServiceField, error) {
	return c.Queries.ListServiceFields(ctx, serviceID)
}

// ListServiceEnvironments returns service environment releases.
func (c *Database) ListServiceEnvironments(ctx context.Context, serviceID int64) ([]queries.ListServiceEnvironmentsRow, error) {
	return c.Queries.ListServiceEnvironments(ctx, serviceID)
}

// ListPendingCommitsNotInProd returns pending commits.
func (c *Database) ListPendingCommitsNotInProd(ctx context.Context, params queries.ListPendingCommitsNotInProdParams) ([]queries.ListPendingCommitsNotInProdRow, error) {
	return c.Queries.ListPendingCommitsNotInProd(ctx, params)
}

// ListDeploymentHistoryByService returns recent deployment history.
func (c *Database) ListDeploymentHistoryByService(ctx context.Context, params queries.ListDeploymentHistoryByServiceParams) ([]queries.ListDeploymentHistoryByServiceRow, error) {
	return c.Queries.ListDeploymentHistoryByService(ctx, params)
}

// MarkServiceIntegrationType updates a service integration type.
func (c *Database) MarkServiceIntegrationType(ctx context.Context, integrationType string, serviceID int64) error {
	return c.Queries.MarkServiceIntegrationType(ctx, queries.MarkServiceIntegrationTypeParams{
		IntegrationType: integrationType,
		ID:              serviceID,
	})
}

// ListOrganizationRequiredFields returns required fields for an org.
func (c *Database) ListOrganizationRequiredFields(ctx context.Context, organizationID int64) ([]queries.ListOrganizationRequiredFieldsRow, error) {
	return c.Queries.ListOrganizationRequiredFields(ctx, organizationID)
}

// DeleteOrganizationRequiredFields removes required fields for an org.
func (c *Database) DeleteOrganizationRequiredFields(ctx context.Context, organizationID int64) error {
	return c.Queries.DeleteOrganizationRequiredFields(ctx, organizationID)
}

// CreateOrganizationRequiredField inserts a required field.
func (c *Database) CreateOrganizationRequiredField(ctx context.Context, params queries.CreateOrganizationRequiredFieldParams) (queries.OrganizationRequiredField, error) {
	return c.Queries.CreateOrganizationRequiredField(ctx, params)
}

// UpdateOrganizationSecrets updates auth token and webhook secret.
func (c *Database) UpdateOrganizationSecrets(ctx context.Context, authToken, webhookSecret string, enabled bool, organizationID int64) error {
	enabledValue := int64(0)
	if enabled {
		enabledValue = 1
	}
	return c.Queries.UpdateOrganizationSecrets(ctx, queries.UpdateOrganizationSecretsParams{
		AuthToken:     authToken,
		WebhookSecret: webhookSecret,
		Enabled:       enabledValue,
		ID:            organizationID,
	})
}

// WithTx runs a function within a transaction.
func (c *Database) WithTx(ctx context.Context, fn func(*queries.Queries) error) error {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	if err := fn(c.Queries.WithTx(tx)); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	return tx.Commit()
}
