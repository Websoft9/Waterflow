package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto migrate
	err = db.AutoMigrate(&WorkflowDefinition{})
	require.NoError(t, err)

	return db
}

func TestDatabaseDefinitionStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	def := &WorkflowDefinition{
		Name:        "test-workflow",
		DisplayName: "Test Workflow",
		Description: "A test workflow",
		Category:    "test",
		Content:     "name: test\njobs:\n  test:\n    runs-on: default",
		ParamsData: []Parameter{
			{Name: "env", Type: "string", Default: "dev"},
		},
	}

	err := store.Create(ctx, def)
	assert.NoError(t, err)
	assert.NotEmpty(t, def.ContentHash)
	assert.False(t, def.CreatedAt.IsZero())
}

func TestDatabaseDefinitionStore_Create_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	def := &WorkflowDefinition{
		Name:    "test-workflow",
		Content: "name: test",
	}

	err := store.Create(ctx, def)
	require.NoError(t, err)

	// Try to create duplicate
	err = store.Create(ctx, def)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestDatabaseDefinitionStore_Get(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	original := &WorkflowDefinition{
		Name:        "test-workflow",
		Description: "Test workflow",
		Content:     "name: test",
	}
	err := store.Create(ctx, original)
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, "test-workflow")
	assert.NoError(t, err)
	assert.Equal(t, "test-workflow", retrieved.Name)
	assert.Equal(t, "Test workflow", retrieved.Description)
}

func TestDatabaseDefinitionStore_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDatabaseDefinitionStore_List(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	// Create multiple definitions
	for i := 1; i <= 5; i++ {
		def := &WorkflowDefinition{
			Name:     string(rune('a'+i-1)) + "-workflow",
			Category: "test",
			Content:  "name: test",
		}
		err := store.Create(ctx, def)
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// List all
	defs, total, err := store.List(ctx, DefinitionFilter{
		Page:  1,
		Limit: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, defs, 5)
}

func TestDatabaseDefinitionStore_List_WithFilter(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	// Create definitions with different categories
	store.Create(ctx, &WorkflowDefinition{Name: "deploy-1", Category: "deployment", Content: "name: test"})
	store.Create(ctx, &WorkflowDefinition{Name: "deploy-2", Category: "deployment", Content: "name: test"})
	store.Create(ctx, &WorkflowDefinition{Name: "test-1", Category: "testing", Content: "name: test"})

	// Filter by category
	defs, total, err := store.List(ctx, DefinitionFilter{
		Category: "deployment",
		Page:     1,
		Limit:    10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, defs, 2)
}

func TestDatabaseDefinitionStore_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	// Create 5 definitions
	for i := 1; i <= 5; i++ {
		def := &WorkflowDefinition{
			Name:    string(rune('a'+i-1)) + "-workflow",
			Content: "name: test",
		}
		store.Create(ctx, def)
		time.Sleep(time.Millisecond)
	}

	// Get page 1
	defs, total, err := store.List(ctx, DefinitionFilter{
		Page:  1,
		Limit: 2,
	})
	assert.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, defs, 2)

	// Get page 2
	defs, total, err = store.List(ctx, DefinitionFilter{
		Page:  2,
		Limit: 2,
	})
	assert.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, defs, 2)
}

func TestDatabaseDefinitionStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	original := &WorkflowDefinition{
		Name:        "test-workflow",
		Description: "Original description",
		Content:     "name: test",
	}
	err := store.Create(ctx, original)
	require.NoError(t, err)

	// Update
	original.Description = "Updated description"
	original.Content = "name: test\nvars:\n  new: value"
	err = store.Update(ctx, original)
	assert.NoError(t, err)

	// Verify
	retrieved, err := store.Get(ctx, "test-workflow")
	assert.NoError(t, err)
	assert.Equal(t, "Updated description", retrieved.Description)
	assert.Contains(t, retrieved.Content, "new: value")
}

func TestDatabaseDefinitionStore_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	def := &WorkflowDefinition{
		Name:    "nonexistent",
		Content: "name: test",
	}
	err := store.Update(ctx, def)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDatabaseDefinitionStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	def := &WorkflowDefinition{
		Name:    "test-workflow",
		Content: "name: test",
	}
	err := store.Create(ctx, def)
	require.NoError(t, err)

	// Delete
	err = store.Delete(ctx, "test-workflow")
	assert.NoError(t, err)

	// Verify deleted
	_, err = store.Get(ctx, "test-workflow")
	assert.Error(t, err)
}

func TestDatabaseDefinitionStore_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDatabaseDefinitionStore_Exists(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)
	ctx := context.Background()

	def := &WorkflowDefinition{
		Name:    "test-workflow",
		Content: "name: test",
	}
	err := store.Create(ctx, def)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "test-workflow")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = store.Exists(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGenerateContentHash(t *testing.T) {
	db := setupTestDB(t)
	store := NewDatabaseDefinitionStore(db)

	hash1 := store.generateContentHash("content1")
	hash2 := store.generateContentHash("content1")
	hash3 := store.generateContentHash("content2")

	assert.Equal(t, hash1, hash2)    // Same content = same hash
	assert.NotEqual(t, hash1, hash3) // Different content = different hash
	assert.Len(t, hash1, 64)         // SHA-256 produces 64 hex characters
}
