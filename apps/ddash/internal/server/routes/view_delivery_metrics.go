package routes

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/views/components"
)

func (v *ViewRoutes) handleLeadTimeMetrics(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	days := 30
	if raw := c.QueryParam("days"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			days = parsed
		}
	}
	report, err := v.read.BuildLeadTimeReport(ctx, orgID, days)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, report)
}

func (v *ViewRoutes) handleServiceMetrics(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	serviceName := c.Param("name")
	if serviceName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "service name required")
	}
	days := 30
	if raw := c.QueryParam("days"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			days = parsed
		}
	}
	metrics, err := v.read.GetServiceMetrics(ctx, orgID, serviceName, days)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, metrics)
}

func (v *ViewRoutes) handleOrgMetrics(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	days := 30
	if raw := c.QueryParam("days"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			days = parsed
		}
	}
	metrics, err := v.read.GetOrgMetrics(ctx, orgID, days)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, metrics)
}

func (v *ViewRoutes) handleServiceMetricsFragment(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	serviceName := c.Param("name")
	if serviceName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "service name required")
	}
	days := 30
	if raw := c.QueryParam("days"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			days = parsed
		}
	}
	metrics, err := v.read.GetServiceMetrics(ctx, orgID, serviceName, days)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", components.ServiceMetricsCard(components.ServiceMetricsData{
		PipelineStats: components.PipelineStatsData{
			PipelineStartedCount:   metrics.PipelineStats.PipelineStartedCount,
			PipelineSucceededCount: metrics.PipelineStats.PipelineSucceededCount,
			PipelineFailedCount:    metrics.PipelineStats.PipelineFailedCount,
			TotalDurationSeconds:   metrics.PipelineStats.TotalDurationSeconds,
			AvgDurationSeconds:     metrics.PipelineStats.AvgDurationSeconds,
		},
		DeploymentDurations: components.DurationStatsData{
			SampleCount:         metrics.DeploymentDurations.SampleCount,
			AvgDurationSeconds:  metrics.DeploymentDurations.AvgDurationSeconds,
			MinDurationSeconds:  metrics.DeploymentDurations.MinDurationSeconds,
			MaxDurationSeconds:  metrics.DeploymentDurations.MaxDurationSeconds,
			LastDurationSeconds: metrics.DeploymentDurations.LastDurationSeconds,
		},
		DriftCount: metrics.DriftCount,
		RedeploymentRate: components.RedeploymentData{
			RedeployCount: metrics.RedeploymentRate.RedeployCount,
			DeployDays:    metrics.RedeploymentRate.DeployDays,
			RedeployRate:  metrics.RedeploymentRate.RedeployRate,
		},
		Throughput: components.ThroughputData{
			WeekStart:        metrics.Throughput.WeekStart,
			ChangesCount:     metrics.Throughput.ChangesCount,
			DeploymentsCount: metrics.Throughput.DeploymentsCount,
		},
		ArtifactAges: func() []components.ArtifactAgeData {
			result := make([]components.ArtifactAgeData, len(metrics.ArtifactAges))
			for i, a := range metrics.ArtifactAges {
				result[i] = components.ArtifactAgeData{
					Environment:   a.Environment,
					ArtifactID:    a.ArtifactID,
					AgeSeconds:    a.AgeSeconds,
					LastEventTsMs: a.LastEventTsMs,
				}
			}
			return result
		}(),
		Comprehensive: components.ComprehensiveData{
			LeadTimeSeconds:              metrics.Comprehensive.LeadTimeSeconds,
			DeploymentFrequency30d:       metrics.Comprehensive.DeploymentFrequency30d,
			ChangeFailureRate:            metrics.Comprehensive.ChangeFailureRate,
			AvgDeploymentDurationSeconds: metrics.Comprehensive.AvgDeploymentDurationSeconds,
			PipelineSuccessCount30d:      metrics.Comprehensive.PipelineSuccessCount30d,
			PipelineFailureCount30d:      metrics.Comprehensive.PipelineFailureCount30d,
			ActiveDeployDays30d:          metrics.Comprehensive.ActiveDeployDays30d,
		},
	}))
}
