package discovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewServiceInstance(t *testing.T) {
	inst := NewServiceInstance("my-service", "10.0.0.1", 8080)

	assert.NotEmpty(t, inst.ID)
	assert.Equal(t, "my-service", inst.ServiceName)
	assert.Equal(t, "10.0.0.1", inst.Host)
	assert.Equal(t, 8080, inst.Port)
	assert.Equal(t, HealthStatusHealthy, inst.HealthStatus)
	assert.Equal(t, 100, inst.Weight)
	assert.NotNil(t, inst.Metadata)
	assert.NotNil(t, inst.Tags)
	assert.False(t, inst.RegisteredAt.IsZero())
	assert.False(t, inst.LastHeartbeat.IsZero())
}

func TestServiceInstance_IsHealthy(t *testing.T) {
	inst := &ServiceInstance{HealthStatus: HealthStatusHealthy}
	assert.True(t, inst.IsHealthy())

	inst.HealthStatus = HealthStatusUnhealthy
	assert.False(t, inst.IsHealthy())

	inst.HealthStatus = HealthStatusUnknown
	assert.False(t, inst.IsHealthy())
}

func TestServiceInstance_IsExpired(t *testing.T) {
	inst := &ServiceInstance{LastHeartbeat: time.Now().UTC().Add(-5 * time.Minute)}
	assert.True(t, inst.IsExpired(3*time.Minute))
	assert.False(t, inst.IsExpired(10*time.Minute))
}

func TestServiceInstance_Address(t *testing.T) {
	inst := &ServiceInstance{Host: "10.0.0.1", Port: 8080}
	assert.Equal(t, "10.0.0.1:8080", inst.Address())

	instZeroPort := &ServiceInstance{Host: "localhost", Port: 0}
	assert.Equal(t, "localhost", instZeroPort.Address())
}

func TestFormatAddress(t *testing.T) {
	assert.Equal(t, "10.0.0.1:80", formatAddress("10.0.0.1", 80))
	assert.Equal(t, "host", formatAddress("host", 0))
}

func TestServiceSet_Count(t *testing.T) {
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "1", HealthStatus: HealthStatusHealthy},
			{ID: "2", HealthStatus: HealthStatusUnhealthy},
		},
	}
	assert.Equal(t, 2, ss.Count())
}

func TestServiceSet_HealthyCount(t *testing.T) {
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "1", HealthStatus: HealthStatusHealthy},
			{ID: "2", HealthStatus: HealthStatusUnhealthy},
			{ID: "3", HealthStatus: HealthStatusHealthy},
		},
	}
	assert.Equal(t, 2, ss.HealthyCount())
}

func TestServiceSet_GetHealthy(t *testing.T) {
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "1", HealthStatus: HealthStatusHealthy},
			{ID: "2", HealthStatus: HealthStatusUnhealthy},
		},
	}
	healthy := ss.GetHealthy()
	assert.Len(t, healthy, 1)
	assert.Equal(t, "1", healthy[0].ID)
}

func TestServiceSet_FilterByZone(t *testing.T) {
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "1", Zone: "us-east-1a"},
			{ID: "2", Zone: "us-west-1a"},
		},
	}
	filtered := ss.FilterByZone("us-east-1a")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "1", filtered[0].ID)
}

func TestServiceSet_FilterByRegion(t *testing.T) {
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "1", Region: "us-east"},
			{ID: "2", Region: "eu-west"},
		},
	}
	filtered := ss.FilterByRegion("eu-west")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "2", filtered[0].ID)
}

func TestServiceQuery_Struct(t *testing.T) {
	q := ServiceQuery{
		ServiceName: "svc",
		Tags:        []string{"v1"},
		Metadata:    map[string]string{"env": "prod"},
		HealthyOnly: true,
		Zone:        "z1",
		Region:      "r1",
	}
	assert.Equal(t, "svc", q.ServiceName)
	assert.True(t, q.HealthyOnly)
}

func TestServiceInstanceStatus_Struct(t *testing.T) {
	s := ServiceInstanceStatus{
		InstanceID:           "i-1",
		ConsecutiveFailures:  2,
		ConsecutiveSuccesses: 5,
		TotalRequests:        100,
		FailedRequests:       3,
	}
	assert.Equal(t, "i-1", s.InstanceID)
	assert.Equal(t, int64(100), s.TotalRequests)
}
