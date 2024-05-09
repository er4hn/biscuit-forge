package dblogic

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// RepoRoleType is a possible repo role
type RepoRoleType int

const (
	// UnknownRole represents an error case
	UnknownRole RepoRoleType = iota
	// OwnerRole repreesnts a repo owner
	OwnerRole
	// ReaderRole represents a repo or repogroup reader
	ReaderRole
	// WriterRole represents a repo or repogroup writer
	WriterRole
)

// UserInGroup represents a user being a member of a usergroup
type UserInGroup struct {
	// UserId is the id of the user
	UserId int
	// UsergroupId is the id of the usergroup
	UsergroupId int
}

// UserGroupInGroup represnts a nested usergroup
type UserGroupInGroup struct {
	// ParentUsergroupId is the ID of the parent usergroup
	ParentUsergroupId int
	// ChildUsergroupId is the ID of the child usergroup
	ChildUsergroupId int
}

// UsergroupRelationships represents the set of relevant
// usergroup relationships for the user.
type UsergroupRelationships struct {
	// UserInGroups is a list of groups the user is a member of.
	UserInGroups []*UserInGroup
	// UserGroupInGroups is a list of nested usergroups
	UserGroupInGroups []*UserGroupInGroup
}

// RepogroupRel represents a relationship between a repogroup and a repo.
type RepogroupRel struct {
	// RepogroupId is the ID of the repogroup
	RepogroupId int
	// RepoId is the ID of the repo
	RepoId int
}

// UserOrGroup specifies if the relationship is for a user or a usergroup
type UserOrGroupRel int

const (
	UndefUGR UserOrGroupRel = iota
	UserUGR
	UsergroupUGR
)

// RepoOrGroup specifies if the relationship is for a repo or a repogroup
type RepoOrGroupRel int

const (
	UndefRGR RepoOrGroupRel = iota
	RepoUGR
	RepogroupUGR
)

// AssignedRole covers a relationship between a role, a user or usergroup, and a repo or repogroup.
type AssignedRole struct {
	// UserOrGroup specifies if UserOrGroupID is a userid or a usergroup id
	UserOrGroup UserOrGroupRel
	// UserOrGroupID is an ID for either a user or usergroup
	UserOrGroupID int
	// RepoOrGroup specifies if RepoOrGroupID is a repoid or a repogroup id
	RepoOrGroup RepoOrGroupRel
	// RepoOrGroupID is an ID for either a repo or repogroup
	RepoOrGroupID int
	// RepoRole is the role the relationship defines
	RepoRole RepoRoleType
}

// RequestDetails provides information about the user logging in.
type RequestDetails struct {
	// UserId is the user id of the user
	UserId int
	// Username is the username of the user
	Username string
	// UsergroupRelationships is the set of relevant usergroup relationships for the authz logic to use in eval.
	UsergroupRelationships *UsergroupRelationships
	// RepoId is the id of the repo being acted upon
	RepoId int
	// RepoName is the name of the repo being acted upon
	RepoName string
	// RepogroupRels is the list of relevant repogroups for authz logic to use in eval.
	RepogroupRels []*RepogroupRel
	// AssignedRoles is the set of roles assigned between entities and repos
	AssignedRoles []*AssignedRole
}

// DBInstance passes around an instance of the pointer to the DB for handling close operations, creating Tx's, etc.
type DBInstance struct {
	// sqliteDb is the open instance of the sqlite db
	sqliteDb *sql.DB
	// filepath is the underlying filepath the db is located at
	filepath string
}

// repoRoleStrToEnum converts the repo role to an enum. Returns an error if unable to
// convert.
func repoRoleStrToEnum(repoRoleStr string, isRepogroup bool) (RepoRoleType, error) {
	switch repoRoleStr {
	case "owner":
		if isRepogroup {
			// This is actually a violation of the underlying sql logic, since there is no owner role in the underlying repogroup by design.
			return UnknownRole, fmt.Errorf("repogroups cannot have owner roles")
		}
		return OwnerRole, nil
	case "reader":
		return ReaderRole, nil
	case "writer":
		return WriterRole, nil
	default:
		return UnknownRole, fmt.Errorf("unable to convert %s", repoRoleStr)
	}
}

// Close closes the underlying database instance.
func (dbInstance *DBInstance) Close() error {
	err := dbInstance.sqliteDb.Close()
	if err != nil {
		return fmt.Errorf("error when closing sqlite db at %s: %w",
			dbInstance.filepath, err)
	}
	return nil
}

// InitDb initializes a sqlite database and fills it with test data
func InitDb() (*DBInstance, error) {
	dbInitFilename := "dblogic/db-init.sql"
	sqlInitBytes, err := os.ReadFile(dbInitFilename)
	if err != nil {
		return nil, fmt.Errorf("error in os.ReadFile for %s: %w",
			dbInitFilename, err)
	}
	sqlInit := string(sqlInitBytes)

	sqliteDbFilename := "forgeAuthz.db"
	file, err := os.Create(sqliteDbFilename)
	if err != nil {
		return nil, fmt.Errorf("error in os.Create for %s: %w",
			sqliteDbFilename, err)
	}
	err = file.Close()
	if err != nil {
		return nil, fmt.Errorf("error when closing %s: %w",
			sqliteDbFilename, err)
	}

	sqliteDb, err := sql.Open("sqlite3", sqliteDbFilename)
	if err != nil {
		return nil, fmt.Errorf("error when trying to open %s: %w",
			sqliteDbFilename, err)
	}

	// Initialize the DB with the example schema and data.
	_, err = sqliteDb.Exec(sqlInit)
	if err != nil {
		return nil, fmt.Errorf("error when trying to init sqlite db: %w",
			err)
	}

	dbInstance := &DBInstance{
		sqliteDb: sqliteDb,
		filepath: sqliteDbFilename,
	}
	return dbInstance, nil
}

// checkUserInDb will check if the user is in the database. If the user is found the username will be returned. An error will be returned if the user cannot be found or other issues occur. sqlTx will not be rolled back by this function if an error occurs.
func checkUserInDb(userId int, sqlTx *sql.Tx) (username string, err error) {
	getUsernameQuery := "SELECT id, username FROM Users WHERE id = $userid"
	var userName string
	sqlRow := sqlTx.QueryRow(getUsernameQuery, sql.Named("userid", userId))
	err = sqlRow.Scan(&userId, &userName)
	if err != nil {
		return "", fmt.Errorf("error querying for user from DB: %w", err)
	}
	return userName, nil
}

// checkRepoInDb will check if the repo is in the database. If the repo is found the repoid will be returned. An error will be returned if the repo cannot be found or other issues occur. sqlTx will not be rolled back by this function if an error occurs.
func checkRepoInDb(reponame string, sqlTx *sql.Tx) (int, error) {
	getReponameQuery := "SELECT id, reponame FROM Repos WHERE reponame = $reponame"
	var repoId int
	sqlRow := sqlTx.QueryRow(getReponameQuery, sql.Named("reponame", reponame))
	err := sqlRow.Scan(&repoId, &reponame)
	if err != nil {
		return 0, fmt.Errorf("error querying for repo from DB: %w", err)
	}
	return repoId, nil
}

// getRepogroupRels will get the set of repogroups that the repo is in. An empty list will be returned if no groups are found. An error will be returned if issues occur. sqlTx will not be rolled back by this function if an error occurs.
func getRepogroupRels(repoId int, sqlTx *sql.Tx) ([]*RepogroupRel, error) {
	getRepogroupQuery := "SELECT repogroup_id FROM RepoGroup_membership WHERE repo_id = $repoid"
	sqlRows, err := sqlTx.Query(getRepogroupQuery, sql.Named("repoid", repoId))
	if err != nil {
		return nil, fmt.Errorf("error when querying for repogroup membership: %w", err)
	}
	defer sqlRows.Close()

	repogroupRels := []*RepogroupRel{}
	for sqlRows.Next() {
		var repogroupId int
		if err := sqlRows.Scan(&repogroupId); err != nil {
			return nil, fmt.Errorf("error when scanning for repogroups: %w", err)
		}
		repogroupRel := &RepogroupRel{
			RepogroupId: repogroupId,
			RepoId:      repoId,
		}
		repogroupRels = append(repogroupRels, repogroupRel)
	}
	sqlRows.Close()

	return repogroupRels, nil
}

// getUsergroupsRecursive will get the set of usergroups the user is in, as well as all child usergroups that the parent usergroup has authority over. An error will be returned if issues occur. sqlTx will not be rolled back by this function if an error occurs.
func getUsergroupsRecursive(userId int, sqlTx *sql.Tx) (*UsergroupRelationships, error) {
	getUsergroupsQuery := `WITH RECURSIVE ugs (
    usergroup_id,
    child_usergroup_id
)
AS (
    SELECT usergroup_id,
           NULL
      FROM UserGroup_membership_users
     WHERE user_id = $userid
    UNION
    SELECT UserGroup_membership_usergroups.usergroup_id,
           UserGroup_membership_usergroups.child_usergroup_id
      FROM UserGroup_membership_usergroups,
           ugs
     WHERE UserGroup_membership_usergroups.usergroup_id = ugs.usergroup_id OR 
           UserGroup_membership_usergroups.usergroup_id = ugs.child_usergroup_id
)
SELECT ugs.usergroup_id,
       ugs.child_usergroup_id
  FROM ugs;
`
	sqlRows, err := sqlTx.Query(getUsergroupsQuery, sql.Named("userid", userId))
	if err != nil {
		return nil, fmt.Errorf("error when querying recursively for usergroups: %w", err)
	}
	defer sqlRows.Close()

	userInGroups := []*UserInGroup{}
	usergroupInGroups := []*UserGroupInGroup{}
	for sqlRows.Next() {
		var usergroupId int
		var childUsergroupId sql.NullInt64
		if err := sqlRows.Scan(&usergroupId, &childUsergroupId); err != nil {
			return nil, fmt.Errorf("error when scanning for usergroups: %w", err)
		}
		if childUsergroupId.Valid {
			// This is a nested usergroup
			// (silly  conversion, okay with panicking here)
			childUgIdInt := int(childUsergroupId.Int64)
			usergroupInGroup := &UserGroupInGroup{
				ParentUsergroupId: usergroupId,
				ChildUsergroupId:  childUgIdInt,
			}
			usergroupInGroups = append(usergroupInGroups, usergroupInGroup)
		} else {
			// This is a usergroup the user is a member of
			userInGroup := &UserInGroup{
				UserId:      userId,
				UsergroupId: usergroupId,
			}
			userInGroups = append(userInGroups, userInGroup)
		}
	}
	sqlRows.Close()

	usergroupRels := &UsergroupRelationships{
		UserInGroups:      userInGroups,
		UserGroupInGroups: usergroupInGroups,
	}
	return usergroupRels, nil
}

// getAssignedRoles will get the set of assigned roles that map users to roles. An empty list will be returned if no mappings are found. An error will be returned if issues occur. sqlTx will  not be rolled back if an error occurs.
func getAssignedRoles(userId int, repoId int, usergroupRels *UsergroupRelationships, repogroupRels []*RepogroupRel, sqlTx *sql.Tx) ([]*AssignedRole, error) {
	assignedRoles := []*AssignedRole{}

	// Get how the user is mapped to repo roles
	getUserRepoRolesQuery := `SELECT repo_roles_enum.rolename
FROM Repo_Roles_membership_Users
INNER JOIN repo_roles_enum
    ON repo_roles_enum.id = Repo_Roles_membership_Users.repo_role
WHERE Repo_Roles_membership_Users.repo_id = $repoid
        AND Repo_Roles_membership_Users.user_id = $userid`
	sqlRows, err := sqlTx.Query(getUserRepoRolesQuery,
		sql.Named("repoid", repoId),
		sql.Named("userid", userId),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting how user is mapped to repo roles: %w", err)
	}
	defer sqlRows.Close()
	for sqlRows.Next() {
		var repoRoleStr string
		if err := sqlRows.Scan(&repoRoleStr); err != nil {
			return nil, fmt.Errorf("error when scanning for repo role str: %w", err)
		}
		repoRole, err := repoRoleStrToEnum(repoRoleStr, false)
		if err != nil {
			return nil, fmt.Errorf("error when mapping db role to enum: %w", err)
		}
		assignedRole := &AssignedRole{
			UserOrGroup:   UserUGR,
			UserOrGroupID: userId,
			RepoOrGroup:   RepoUGR,
			RepoOrGroupID: repoId,
			RepoRole:      repoRole,
		}
		assignedRoles = append(assignedRoles, assignedRole)
	}
	sqlRows.Close()

	// Get the set of usergroups mapped to repo roles
	// Build all of the usergroups of interest into one big query.
	getUsergroupRepoRolesQuery := `SELECT Repo_Roles_membership_Usergroups.usergroup_id,
         repo_roles_enum.rolename
FROM Repo_Roles_membership_UserGroups
INNER JOIN repo_roles_enum
    ON repo_roles_enum.id = Repo_Roles_membership_UserGroups.repo_role
WHERE Repo_Roles_membership_Usergroups.repo_id = $repoid
        AND Repo_Roles_membership_Usergroups.usergroup_id IN ( %s )`
	// treat the set of usergroupIds as a map to avoid issues where the same
	// usergroup is added multiple times.
	usergroupIds := map[int]bool{}
	for _, userInGroups := range usergroupRels.UserInGroups {
		usergroupIds[userInGroups.UsergroupId] = true
	}
	for _, userGroupInGroup := range usergroupRels.UserGroupInGroups {
		usergroupIds[userGroupInGroup.ChildUsergroupId] = true
	}
	usergroupBindList := []string{}
	for _, _ = range usergroupIds {
		usergroupBindList = append(usergroupBindList, "?")
	}
	getUsergroupRepoRolesQuery = fmt.Sprintf(getUsergroupRepoRolesQuery, strings.Join(usergroupBindList, ", "))
	sqlBindParams := []interface{}{sql.Named("repoid", repoId)}
	for ugId, _ := range usergroupIds {
		sqlBindParams = append(sqlBindParams, ugId)
	}
	sqlRows, err = sqlTx.Query(getUsergroupRepoRolesQuery, sqlBindParams...)
	if err != nil {
		return nil, fmt.Errorf("error when querying for usergroup to repo role mappings: %w", err)
	}
	defer sqlRows.Close()
	for sqlRows.Next() {
		var usergroupId int
		var roleNameStr string
		if err := sqlRows.Scan(&usergroupId, &roleNameStr); err != nil {
			return nil, fmt.Errorf("error when scanning for ug / repo role str: %w", err)
		}
		repoRole, err := repoRoleStrToEnum(roleNameStr, false)
		if err != nil {
			return nil, fmt.Errorf("error when mapping db role to enum: %w", err)
		}
		assignedRole := &AssignedRole{
			UserOrGroup:   UsergroupUGR,
			UserOrGroupID: usergroupId,
			RepoOrGroup:   RepoUGR,
			RepoOrGroupID: repoId,
			RepoRole:      repoRole,
		}
		assignedRoles = append(assignedRoles, assignedRole)
	}
	sqlRows.Close()

	// Get the set of users mapped to repogroup roles
	getUserRepogroupRolesQuery := `SELECT RepoGroup_Roles_membership_Users.repogroup_id,
         repogroup_roles_enum.rolename
FROM RepoGroup_Roles_membership_Users
INNER JOIN repogroup_roles_enum
    ON RepoGroup_roles_membership_Users.repogroup_role = repogroup_roles_enum.id
WHERE RepoGroup_roles_membership_Users.user_id = $userid
        AND RepoGroup_Roles_membership_Users.repogroup_id IN ( %s )`
	// treat the set of repogroupIds as a map to avoid issues where the same repogroup
	// is added multiple times.
	repogroupIds := map[int]bool{}
	for _, repogroupRel := range repogroupRels {
		repogroupIds[repogroupRel.RepogroupId] = true
	}
	repogroupBindList := []string{}
	for _, _ = range repogroupIds {
		repogroupBindList = append(repogroupBindList, "?")
	}
	getUserRepogroupRolesQuery = fmt.Sprintf(getUserRepogroupRolesQuery, strings.Join(repogroupBindList, ", "))
	sqlBindParams = []interface{}{sql.Named("userid", userId)}
	for rgId, _ := range repogroupIds {
		sqlBindParams = append(sqlBindParams, rgId)
	}
	sqlRows, err = sqlTx.Query(getUserRepogroupRolesQuery, sqlBindParams...)
	if err != nil {
		return nil, fmt.Errorf("error when querying for user to repogroup role mappings: %w", err)
	}
	defer sqlRows.Close()
	for sqlRows.Next() {
		var repogroupId int
		var roleNameStr string
		if err := sqlRows.Scan(&repogroupId, &roleNameStr); err != nil {
			return nil, fmt.Errorf("error when scanning for repogroup / repo role str: %w", err)
		}
		repoRole, err := repoRoleStrToEnum(roleNameStr, true)
		if err != nil {
			return nil, fmt.Errorf("error when mapping db role to enum: %w", err)
		}
		assignedRole := &AssignedRole{
			UserOrGroup:   UserUGR,
			UserOrGroupID: userId,
			RepoOrGroup:   RepogroupUGR,
			RepoOrGroupID: repogroupId,
			RepoRole:      repoRole,
		}
		assignedRoles = append(assignedRoles, assignedRole)
	}
	sqlRows.Close()

	// Get the set of usergroups mapped to repogroup roles
	getUsergroupRepogroupRolesQuery := `SELECT RepoGroup_Roles_membership_Usergroup.repogroup_id,
         RepoGroup_Roles_membership_Usergroup.usergroup_id,
         repogroup_roles_enum.rolename
FROM RepoGroup_Roles_membership_Usergroup
INNER JOIN repogroup_roles_enum
    ON RepoGroup_Roles_membership_Usergroup.repogroup_role = repogroup_roles_enum.id
WHERE RepoGroup_Roles_membership_Usergroup.usergroup_id IN ( %s)
        AND RepoGroup_Roles_membership_Usergroup.repogroup_id IN ( %s )`
	getUsergroupRepogroupRolesQuery = fmt.Sprintf(getUsergroupRepogroupRolesQuery,
		strings.Join(usergroupBindList, ", "),
		strings.Join(repogroupBindList, ", "),
	)
	sqlBindParams = []interface{}{}
	for ugId, _ := range usergroupIds {
		sqlBindParams = append(sqlBindParams, ugId)
	}
	for rgId, _ := range repogroupIds {
		sqlBindParams = append(sqlBindParams, rgId)
	}
	sqlRows, err = sqlTx.Query(getUsergroupRepogroupRolesQuery, sqlBindParams...)
	if err != nil {
		return nil, fmt.Errorf("error when querying for usergroup to repogroup role mappings: %w", err)
	}
	defer sqlRows.Close()
	for sqlRows.Next() {
		var repogroupId int
		var usergroupId int
		var roleNameStr string
		if err := sqlRows.Scan(&repogroupId, &usergroupId, &roleNameStr); err != nil {
			return nil, fmt.Errorf("error when scanning for repogroup / usergroup / repo role str: %w", err)
		}
		repoRole, err := repoRoleStrToEnum(roleNameStr, true)
		if err != nil {
			return nil, fmt.Errorf("error when mapping db role to enum: %w", err)
		}
		assignedRole := &AssignedRole{
			UserOrGroup:   UsergroupUGR,
			UserOrGroupID: usergroupId,
			RepoOrGroup:   RepogroupUGR,
			RepoOrGroupID: repogroupId,
			RepoRole:      repoRole,
		}
		assignedRoles = append(assignedRoles, assignedRole)
	}
	sqlRows.Close()

	return assignedRoles, nil
}

// GatherRequestDetails provides information about the request for use in authZ.
func GatherRequestDetails(userId int, reponame string, dbInstance *DBInstance) (*RequestDetails, error) {
	reqDetails := &RequestDetails{
		UserId:   userId,
		RepoName: reponame,
	}

	// Since this involves multiple queries a SQL transaction is used to ensure a consistent view of the data until all the queries are done.
	sqlTx, err := dbInstance.sqliteDb.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("error when making Tx: %w", err)
	}

	username, err := checkUserInDb(userId, sqlTx)
	if err != nil {
		sqlTx.Rollback()
		return nil, fmt.Errorf("error from checkUserInDb: %w", err)
	}
	reqDetails.Username = username

	usergroupRels, err := getUsergroupsRecursive(userId, sqlTx)
	if err != nil {
		sqlTx.Rollback()
		return nil, fmt.Errorf("error from getUsergroupsRecursive: %w", err)
	}
	reqDetails.UsergroupRelationships = usergroupRels

	repoId, err := checkRepoInDb(reponame, sqlTx)
	if err != nil {
		sqlTx.Rollback()
		return nil, fmt.Errorf("error from checkRepoInDb: %w", err)
	}
	reqDetails.RepoId = repoId

	repogroupRels, err := getRepogroupRels(repoId, sqlTx)
	if err != nil {
		sqlTx.Rollback()
		return nil, fmt.Errorf("error from getRepogroupRels: %w", err)
	}
	reqDetails.RepogroupRels = repogroupRels

	assignedRoles, err := getAssignedRoles(userId, repoId, usergroupRels, repogroupRels, sqlTx)
	if err != nil {
		sqlTx.Rollback()
		return nil, fmt.Errorf("error from getAssignedRoles: %w", err)
	}
	reqDetails.AssignedRoles = assignedRoles

	// I've seen various conflicting thoughts for if commit or rollback should be used for Tx's intended to be read only. Going with commit since it feels like less of an "error case flow".
	err = sqlTx.Commit()
	if err != nil {
		return nil, fmt.Errorf("error when cleaning up sqlite Tx: %w", err)
	}

	return reqDetails, nil
}
