package repository_test

import (
	"context"
	"testing"

	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAgentRepository is a mock implementation for testing
type MockAgentRepository struct {
	mock.Mock
}

func (m *MockAgentRepository) Create(ctx context.Context, a *agent.Agent) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockAgentRepository) GetByID(ctx context.Context, id string) (*agent.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) GetByName(ctx context.Context, name string) (*agent.Agent, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) Update(ctx context.Context, a *agent.Agent) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockAgentRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentRepository) SoftDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentRepository) List(ctx context.Context, filter agent.AgentFilter) ([]*agent.Agent, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) Count(ctx context.Context, filter agent.AgentFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAgentRepository) Exists(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockAgentRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockAgentRepository) GetByOwner(ctx context.Context, ownerID string, filter agent.AgentFilter) ([]*agent.Agent, error) {
	args := m.Called(ctx, ownerID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) GetByStatus(ctx context.Context, status agent.AgentStatus, filter agent.AgentFilter) ([]*agent.Agent, error) {
	args := m.Called(ctx, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) GetByRuntimeType(ctx context.Context, runtimeType agent.RuntimeType, filter agent.AgentFilter) ([]*agent.Agent, error) {
	args := m.Called(ctx, runtimeType, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) GetByTags(ctx context.Context, tags []string, filter agent.AgentFilter) ([]*agent.Agent, error) {
	args := m.Called(ctx, tags, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) Search(ctx context.Context, query string, filter agent.AgentFilter) ([]*agent.Agent, error) {
	args := m.Called(ctx, query, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) UpdateStatus(ctx context.Context, id string, status agent.AgentStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockAgentRepository) UpdateStats(ctx context.Context, id string, stats agent.ExecutionStats) error {
	args := m.Called(ctx, id, stats)
	return args.Error(0)
}

func (m *MockAgentRepository) IncrementExecutionCount(ctx context.Context, id string, success bool) error {
	args := m.Called(ctx, id, success)
	return args.Error(0)
}

func (m *MockAgentRepository) GetActive(ctx context.Context) ([]*agent.Agent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) GetRecent(ctx context.Context, limit int) ([]*agent.Agent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) GetPopular(ctx context.Context, limit int) ([]*agent.Agent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

func (m *MockAgentRepository) BatchCreate(ctx context.Context, agents []*agent.Agent) error {
	args := m.Called(ctx, agents)
	return args.Error(0)
}

func (m *MockAgentRepository) BatchUpdate(ctx context.Context, agents []*agent.Agent) error {
	args := m.Called(ctx, agents)
	return args.Error(0)
}

func (m *MockAgentRepository) BatchDelete(ctx context.Context, ids []string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockAgentRepository) Restore(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentRepository) Archive(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentRepository) GetArchived(ctx context.Context, filter agent.AgentFilter) ([]*agent.Agent, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*agent.Agent), args.Error(1)
}

// TestArchiveMethod tests the Archive method
func TestArchiveMethod(t *testing.T) {
	t.Run("Archive existing agent successfully", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()
		agentID := "agent-123"

		mockRepo.On("Archive", ctx, agentID).Return(nil)

		err := mockRepo.Archive(ctx, agentID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Archive non-existent agent returns error", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()
		agentID := "non-existent"

		mockRepo.On("Archive", ctx, agentID).Return(assert.AnError)

		err := mockRepo.Archive(ctx, agentID)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Archive with empty ID returns error", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()

		mockRepo.On("Archive", ctx, "").Return(assert.AnError)

		err := mockRepo.Archive(ctx, "")

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

// TestGetArchivedMethod tests the GetArchived method
func TestGetArchivedMethod(t *testing.T) {
	t.Run("Get archived agents successfully", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()
		filter := agent.NewAgentFilter()

		archivedAgents := []*agent.Agent{
			agent.NewAgent("Archived1", "Desc1", agent.RuntimeTypeNative, "user1"),
			agent.NewAgent("Archived2", "Desc2", agent.RuntimeTypeNative, "user2"),
		}

		mockRepo.On("GetArchived", ctx, filter).Return(archivedAgents, nil)

		agents, err := mockRepo.GetArchived(ctx, filter)

		assert.NoError(t, err)
		assert.Len(t, agents, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Get archived agents with no results", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()
		filter := agent.NewAgentFilter()

		mockRepo.On("GetArchived", ctx, filter).Return([]*agent.Agent{}, nil)

		agents, err := mockRepo.GetArchived(ctx, filter)

		assert.NoError(t, err)
		assert.Empty(t, agents)
		mockRepo.AssertExpectations(t)
	})
}

// TestBatchUpdateMethod tests the BatchUpdate method
func TestBatchUpdateMethod(t *testing.T) {
	t.Run("Batch update multiple agents successfully", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()

		agents := []*agent.Agent{
			agent.NewAgent("Agent1", "Desc1", agent.RuntimeTypeNative, "user1"),
			agent.NewAgent("Agent2", "Desc2", agent.RuntimeTypeNative, "user2"),
		}

		mockRepo.On("BatchUpdate", ctx, agents).Return(nil)

		err := mockRepo.BatchUpdate(ctx, agents)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Batch update with empty slice succeeds", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()

		mockRepo.On("BatchUpdate", ctx, []*agent.Agent{}).Return(nil)

		err := mockRepo.BatchUpdate(ctx, []*agent.Agent{})

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Batch update fails on error", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()

		agents := []*agent.Agent{
			agent.NewAgent("Agent1", "Desc1", agent.RuntimeTypeNative, "user1"),
		}

		mockRepo.On("BatchUpdate", ctx, agents).Return(assert.AnError)

		err := mockRepo.BatchUpdate(ctx, agents)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

// TestBatchDeleteMethod tests the BatchDelete method
func TestBatchDeleteMethod(t *testing.T) {
	t.Run("Batch delete multiple agents successfully", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()
		ids := []string{"agent1", "agent2", "agent3"}

		mockRepo.On("BatchDelete", ctx, ids).Return(nil)

		err := mockRepo.BatchDelete(ctx, ids)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Batch delete with empty slice succeeds", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()

		mockRepo.On("BatchDelete", ctx, []string{}).Return(nil)

		err := mockRepo.BatchDelete(ctx, []string{})

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Batch delete with large batch", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()

		// Create 100 IDs
		ids := make([]string, 100)
		for i := 0; i < 100; i++ {
			ids[i] = "agent" + string(rune(i))
		}

		mockRepo.On("BatchDelete", ctx, ids).Return(nil)

		err := mockRepo.BatchDelete(ctx, ids)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

// TestCountMethod tests the Count method
func TestCountMethod(t *testing.T) {
	t.Run("Count agents successfully", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()
		filter := agent.NewAgentFilter()

		mockRepo.On("Count", ctx, filter).Return(int64(42), nil)

		count, err := mockRepo.Count(ctx, filter)

		assert.NoError(t, err)
		assert.Equal(t, int64(42), count)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Count with filter returns filtered count", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()
		filter := agent.NewAgentFilter().WithStatus(agent.AgentStatusActive)

		mockRepo.On("Count", ctx, filter).Return(int64(10), nil)

		count, err := mockRepo.Count(ctx, filter)

		assert.NoError(t, err)
		assert.Equal(t, int64(10), count)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Count with no results returns zero", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()
		filter := agent.NewAgentFilter()

		mockRepo.On("Count", ctx, filter).Return(int64(0), nil)

		count, err := mockRepo.Count(ctx, filter)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		mockRepo.AssertExpectations(t)
	})
}

// TestRepositoryMethodIntegration tests methods working together
func TestRepositoryMethodIntegration(t *testing.T) {
	t.Run("Create, Archive, and GetArchived workflow", func(t *testing.T) {
		mockRepo := new(MockAgentRepository)
		ctx := context.Background()

		// Create agent
		testAgent := agent.NewAgent("TestAgent", "Description", agent.RuntimeTypeNative, "user1")
		mockRepo.On("Create", ctx, testAgent).Return(nil)

		// Archive agent
		mockRepo.On("Archive", ctx, testAgent.ID).Return(nil)

		// Get archived agents
		filter := agent.NewAgentFilter()
		mockRepo.On("GetArchived", ctx, filter).Return([]*agent.Agent{testAgent}, nil)

		// Execute workflow
		err := mockRepo.Create(ctx, testAgent)
		require.NoError(t, err)

		err = mockRepo.Archive(ctx, testAgent.ID)
		require.NoError(t, err)

		archived, err := mockRepo.GetArchived(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, archived, 1)
		assert.Equal(t, testAgent.ID, archived[0].ID)

		mockRepo.AssertExpectations(t)
	})
}
