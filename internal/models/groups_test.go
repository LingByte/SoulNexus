package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGroupsTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&User{}, &Group{}, &GroupMember{}, &GroupInvitation{})
	require.NoError(t, err)

	return db
}

func createTestUserForGroups(t *testing.T, db *gorm.DB, email, username string) *User {
	user := &User{
		Email:    email,
		Password: "hashedpassword",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// GroupPermission tests are in base_test.go to avoid duplication

func TestGroup_CRUD(t *testing.T) {
	db := setupGroupsTestDB(t)
	creator := createTestUserForGroups(t, db, "creator@example.com", "creator")

	// 测试创建组织
	group := &Group{
		Name:      "Test Organization",
		Type:      "company",
		Extra:     "Extra info",
		Avatar:    "https://example.com/avatar.jpg",
		CreatorID: creator.ID,
		Permission: GroupPermission{
			Permissions: []string{"read", "write", "admin"},
		},
	}

	err := db.Create(group).Error
	assert.NoError(t, err)
	assert.NotZero(t, group.ID)
	assert.NotZero(t, group.CreatedAt)

	// 测试读取组织
	var retrieved Group
	err = db.Preload("Creator").First(&retrieved, group.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, group.Name, retrieved.Name)
	assert.Equal(t, group.Type, retrieved.Type)
	assert.Equal(t, creator.ID, retrieved.CreatorID)
	assert.Equal(t, creator.Email, retrieved.Creator.Email)
	assert.Len(t, retrieved.Permission.Permissions, 3)

	// 测试更新组织
	retrieved.Name = "Updated Organization"
	retrieved.Permission = GroupPermission{
		Permissions: []string{"read", "write"},
	}
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated Group
	err = db.First(&updated, group.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Updated Organization", updated.Name)
	assert.Len(t, updated.Permission.Permissions, 2)

	// 测试删除组织
	err = db.Delete(&updated).Error
	assert.NoError(t, err)

	var deleted Group
	err = db.First(&deleted, group.ID).Error
	assert.Error(t, err)
}

func TestGroupMember_CRUD(t *testing.T) {
	db := setupGroupsTestDB(t)
	creator := createTestUserForGroups(t, db, "creator@example.com", "creator")
	member := createTestUserForGroups(t, db, "member@example.com", "member")

	// 创建组织
	group := &Group{
		Name:      "Test Organization",
		Type:      "company",
		CreatorID: creator.ID,
	}
	err := db.Create(group).Error
	require.NoError(t, err)

	// 测试创建组织成员
	groupMember := &GroupMember{
		UserID:  member.ID,
		GroupID: group.ID,
		Role:    "member",
	}

	err = db.Create(groupMember).Error
	assert.NoError(t, err)
	assert.NotZero(t, groupMember.ID)
	assert.NotZero(t, groupMember.CreatedAt)

	// 测试读取组织成员
	var retrieved GroupMember
	err = db.Preload("User").Preload("Group").First(&retrieved, groupMember.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, member.ID, retrieved.UserID)
	assert.Equal(t, group.ID, retrieved.GroupID)
	assert.Equal(t, "member", retrieved.Role)
	assert.Equal(t, member.Email, retrieved.User.Email)
	assert.Equal(t, group.Name, retrieved.Group.Name)

	// 测试更新成员角色
	retrieved.Role = "admin"
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated GroupMember
	err = db.First(&updated, groupMember.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "admin", updated.Role)

	// 测试删除成员
	err = db.Delete(&updated).Error
	assert.NoError(t, err)

	var deleted GroupMember
	err = db.First(&deleted, groupMember.ID).Error
	assert.Error(t, err)
}

func TestGroupInvitation_CRUD(t *testing.T) {
	db := setupGroupsTestDB(t)
	inviter := createTestUserForGroups(t, db, "inviter@example.com", "inviter")
	invitee := createTestUserForGroups(t, db, "invitee@example.com", "invitee")

	// 创建组织
	group := &Group{
		Name:      "Test Organization",
		Type:      "company",
		CreatorID: inviter.ID,
	}
	err := db.Create(group).Error
	require.NoError(t, err)

	// 测试创建邀请
	expiresAt := time.Now().Add(24 * time.Hour)
	invitation := &GroupInvitation{
		GroupID:   group.ID,
		InviterID: inviter.ID,
		InviteeID: invitee.ID,
		Status:    "pending",
		ExpiresAt: &expiresAt,
	}

	err = db.Create(invitation).Error
	assert.NoError(t, err)
	assert.NotZero(t, invitation.ID)
	assert.NotZero(t, invitation.CreatedAt)

	// 测试读取邀请
	var retrieved GroupInvitation
	err = db.Preload("Group").Preload("Inviter").Preload("Invitee").First(&retrieved, invitation.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, group.ID, retrieved.GroupID)
	assert.Equal(t, inviter.ID, retrieved.InviterID)
	assert.Equal(t, invitee.ID, retrieved.InviteeID)
	assert.Equal(t, "pending", retrieved.Status)
	assert.Equal(t, group.Name, retrieved.Group.Name)
	assert.Equal(t, inviter.Email, retrieved.Inviter.Email)
	assert.Equal(t, invitee.Email, retrieved.Invitee.Email)

	// 测试更新邀请状态
	retrieved.Status = "accepted"
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated GroupInvitation
	err = db.First(&updated, invitation.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "accepted", updated.Status)

	// 测试删除邀请
	err = db.Delete(&updated).Error
	assert.NoError(t, err)

	var deleted GroupInvitation
	err = db.First(&deleted, invitation.ID).Error
	assert.Error(t, err)
}

func TestGroupInvitation_StatusTransitions(t *testing.T) {
	db := setupGroupsTestDB(t)
	inviter := createTestUserForGroups(t, db, "inviter@example.com", "inviter")
	invitee := createTestUserForGroups(t, db, "invitee@example.com", "invitee")

	group := &Group{
		Name:      "Test Organization",
		Type:      "company",
		CreatorID: inviter.ID,
	}
	err := db.Create(group).Error
	require.NoError(t, err)

	// 测试不同的状态转换
	statuses := []string{"pending", "accepted", "rejected"}

	for _, status := range statuses {
		invitation := &GroupInvitation{
			GroupID:   group.ID,
			InviterID: inviter.ID,
			InviteeID: invitee.ID,
			Status:    status,
		}

		err = db.Create(invitation).Error
		assert.NoError(t, err)

		var retrieved GroupInvitation
		err = db.First(&retrieved, invitation.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, status, retrieved.Status)
	}
}

func TestGroupMember_Roles(t *testing.T) {
	db := setupGroupsTestDB(t)
	creator := createTestUserForGroups(t, db, "creator@example.com", "creator")
	user := createTestUserForGroups(t, db, "user@example.com", "user")

	group := &Group{
		Name:      "Test Organization",
		Type:      "company",
		CreatorID: creator.ID,
	}
	err := db.Create(group).Error
	require.NoError(t, err)

	// 测试不同角色
	roles := []string{"member", "admin", "owner", "viewer"}

	for i, role := range roles {
		member := &GroupMember{
			UserID:  user.ID,
			GroupID: group.ID,
			Role:    role,
		}

		err = db.Create(member).Error
		assert.NoError(t, err)

		var retrieved GroupMember
		err = db.First(&retrieved, member.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, role, retrieved.Role)

		// 清理，为下一个测试准备
		if i < len(roles)-1 {
			db.Delete(&retrieved)
		}
	}
}

func TestGroup_WithComplexPermissions(t *testing.T) {
	db := setupGroupsTestDB(t)
	creator := createTestUserForGroups(t, db, "creator@example.com", "creator")

	// 测试复杂权限结构
	complexPermissions := GroupPermission{
		Permissions: []string{
			"users.read", "users.write", "users.delete",
			"groups.read", "groups.write",
			"devices.read", "devices.write", "devices.admin",
			"assistants.read",
		},
	}

	group := &Group{
		Name:       "Complex Permissions Group",
		Type:       "enterprise",
		CreatorID:  creator.ID,
		Permission: complexPermissions,
	}

	err := db.Create(group).Error
	assert.NoError(t, err)

	var retrieved Group
	err = db.First(&retrieved, group.ID).Error
	assert.NoError(t, err)
	assert.Len(t, retrieved.Permission.Permissions, 9)
	assert.Contains(t, retrieved.Permission.Permissions, "users.read")
	assert.Contains(t, retrieved.Permission.Permissions, "devices.admin")
	assert.Contains(t, retrieved.Permission.Permissions, "assistants.read")
}

func TestGroupInvitation_ExpirationHandling(t *testing.T) {
	db := setupGroupsTestDB(t)
	inviter := createTestUserForGroups(t, db, "inviter@example.com", "inviter")
	invitee := createTestUserForGroups(t, db, "invitee@example.com", "invitee")

	group := &Group{
		Name:      "Test Organization",
		Type:      "company",
		CreatorID: inviter.ID,
	}
	err := db.Create(group).Error
	require.NoError(t, err)

	// 测试已过期的邀请
	pastTime := time.Now().Add(-24 * time.Hour)
	expiredInvitation := &GroupInvitation{
		GroupID:   group.ID,
		InviterID: inviter.ID,
		InviteeID: invitee.ID,
		Status:    "pending",
		ExpiresAt: &pastTime,
	}

	err = db.Create(expiredInvitation).Error
	assert.NoError(t, err)

	// 测试未来过期的邀请
	futureTime := time.Now().Add(24 * time.Hour)
	validInvitation := &GroupInvitation{
		GroupID:   group.ID,
		InviterID: inviter.ID,
		InviteeID: invitee.ID,
		Status:    "pending",
		ExpiresAt: &futureTime,
	}

	err = db.Create(validInvitation).Error
	assert.NoError(t, err)

	// 查询过期的邀请
	var expiredInvitations []GroupInvitation
	err = db.Where("expires_at < ? AND status = ?", time.Now(), "pending").Find(&expiredInvitations).Error
	assert.NoError(t, err)
	assert.Len(t, expiredInvitations, 1)
	assert.Equal(t, expiredInvitation.ID, expiredInvitations[0].ID)

	// 查询有效的邀请
	var validInvitations []GroupInvitation
	err = db.Where("expires_at > ? AND status = ?", time.Now(), "pending").Find(&validInvitations).Error
	assert.NoError(t, err)
	assert.Len(t, validInvitations, 1)
	assert.Equal(t, validInvitation.ID, validInvitations[0].ID)
}

func TestGroup_QueryByType(t *testing.T) {
	db := setupGroupsTestDB(t)
	creator := createTestUserForGroups(t, db, "creator@example.com", "creator")

	// 创建不同类型的组织
	types := []string{"company", "team", "project", "department"}

	for _, groupType := range types {
		group := &Group{
			Name:      "Test " + groupType,
			Type:      groupType,
			CreatorID: creator.ID,
		}
		err := db.Create(group).Error
		require.NoError(t, err)
	}

	// 按类型查询
	var companies []Group
	err := db.Where("type = ?", "company").Find(&companies).Error
	assert.NoError(t, err)
	assert.Len(t, companies, 1)
	assert.Equal(t, "Test company", companies[0].Name)

	// 查询所有组织
	var allGroups []Group
	err = db.Find(&allGroups).Error
	assert.NoError(t, err)
	assert.Len(t, allGroups, 4)
}

// Benchmark tests
func BenchmarkGroupPermission_Value(b *testing.B) {
	gp := GroupPermission{
		Permissions: []string{"read", "write", "admin", "delete", "create"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gp.Value()
	}
}

func BenchmarkGroupPermission_Scan(b *testing.B) {
	data := []byte(`{"Permissions":["read","write","admin","delete","create"]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var gp GroupPermission
		gp.Scan(data)
	}
}

func BenchmarkGroup_Create(b *testing.B) {
	db := setupGroupsTestDB(&testing.T{})
	creator := createTestUserForGroups(&testing.T{}, db, "creator@example.com", "creator")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		group := &Group{
			Name:      "Benchmark Group " + string(rune(i)),
			Type:      "company",
			CreatorID: creator.ID,
			Permission: GroupPermission{
				Permissions: []string{"read", "write"},
			},
		}
		db.Create(group)
	}
}
