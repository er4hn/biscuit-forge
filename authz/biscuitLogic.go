package authz

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/biscuit-auth/biscuit-go/v2"
	"github.com/biscuit-auth/biscuit-go/v2/parser"

	"biscuitExample/dblogic"
)

type Action int

const (
	Membership Action = iota
	Read
	Write
)

const (
	membershipStr = "membership"
	writeStr      = "write"
	readStr       = "read"
	ownerRoleStr  = "owner"
	writerRoleStr = "writer"
	readerRoleStr = "reader"
)

const (
	roleNS      = "role"
	actionNS    = "action"
	userNS      = "userid"
	usergroupNS = "usergroupid"
	repoNS      = "repo"
	repogroupNS = "repogroupid"
)

// TokenIssuer issues a biscuit with a user's token.
// NOTE: This is example code and in the real world keep private keys tightly accessc controlled.
type TokenIssuer struct {
	privateRoot ed25519.PrivateKey
	PublicRoot  ed25519.PublicKey
}

// IssueToken issues a biscuit for a user in string format.
func (tokenIssuer *TokenIssuer) IssueToken(userId int) (*biscuit.Biscuit, error) {
	// There is a function FromStringBlockWithParams, but I can't get it
	// to work properly
	// (below just results in {id} not being filled in)
	/*
		authority, err := parser.FromStringBlockWithParams(
			`user("userid:{id}");`,
			map[string]biscuit.Term{
				"id": biscuit.Integer(userId),
			},
		)
	*/
	userFact := fmt.Sprintf(`user("userid:%d");`, userId)
	authority, err := parser.FromStringBlock(userFact)
	if err != nil {
		return nil, fmt.Errorf("error when parsing authority block: %w",
			err)
	}

	builder := biscuit.NewBuilder(tokenIssuer.privateRoot)
	err = builder.AddBlock(authority)
	if err != nil {
		return nil, fmt.Errorf("error when adding authority block: %w",
			err)
	}

	biscuitToken, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("error when building biscuit: %w", err)
	}
	return biscuitToken, nil
}

// NewTokenIssuer creates and returns a new TokenIssuer to create biscuits.
func NewTokenIssuer() (*TokenIssuer, error) {
	publicRoot, privateRoot, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("error generating new RoT for token issuer: %w",
			err)
	}

	tokenIssuer := &TokenIssuer{
		privateRoot: privateRoot,
		PublicRoot:  publicRoot,
	}
	return tokenIssuer, nil
}

// namespaceAuthz adds a namespace to a datalog symbol
func namespaceAuthz(symbol string, namespace string) string {
	namespacedStr := fmt.Sprintf("%s:%s", namespace, symbol)
	return namespacedStr
}

// namespaceRole is a special case of namespaceAuthz for roleNS
func namespaceRole(symbol string) string {
	namespacedStr := namespaceAuthz(symbol, roleNS)
	return namespacedStr
}

// namespaceUser is a special case of namespaceAuthz for userNS
func namespaceUser(symbol int) string {
	symbolStr := fmt.Sprintf("%d", symbol)
	namespacedStr := namespaceAuthz(symbolStr, userNS)
	return namespacedStr
}

// namespaceRepo is a special case of namespaceAuthz for repoNS
func namespaceRepo(symbol int) string {
	symbolStr := fmt.Sprintf("%d", symbol)
	namespacedStr := namespaceAuthz(symbolStr, repoNS)
	return namespacedStr
}

// namespaceAction is a special case of namespaceAuthz for actionNS
func namespaceAction(symbol string) string {
	namespacedStr := namespaceAuthz(symbol, actionNS)
	return namespacedStr
}

// namespaceUG is a special case of namespaceAuthz for usergroupNS
func namespaceUG(symbol int) string {
	symbolStr := fmt.Sprintf("%d", symbol)
	namespacedStr := namespaceAuthz(symbolStr, usergroupNS)
	return namespacedStr
}

// namespaceRG is a space case of namespaceAuthz for repogroupNS
func namespaceRG(symbol int) string {
	symbolStr := fmt.Sprintf("%d", symbol)
	namespacedStr := namespaceAuthz(symbolStr, repogroupNS)
	return namespacedStr
}

// buildRoleActions turns a set of actions into the datalog values for repo_role_actions
func buildRoleActions(actions []string) string {
	datalogedActions := []string{}
	for _, action := range actions {
		namespacedAction := namespaceAuthz(action, actionNS)
		quoteAction := fmt.Sprintf(`"%s"`, namespacedAction)
		datalogedActions = append(datalogedActions, quoteAction)
	}
	joinedActions := strings.Join(datalogedActions, ", ")
	return joinedActions
}

// CheckAuthz decides if the user in userDetails has permission to perform operation against repo.
func CheckAuthz(token *biscuit.Biscuit, publicRoot ed25519.PublicKey, reqDetails *dblogic.RequestDetails, operation Action) (bool, error) {
	type repoRoleActions struct {
		RoleName           string
		RoleAllowedActions []string
	}
	type userInGroup struct {
		UsergroupId int
		UserId      int
	}
	type ugInUg struct {
		ParentUsergroupId int
		ChildUsergroupId  int
	}
	type userGroupRels struct {
		UserInGroups []*userInGroup
		UgsInUgs     []*ugInUg
	}
	type assignedRole struct {
		Role           string
		UserNamespaced string
		RepoNamespaced string
	}
	type authzDetails struct {
		RepoRoleActions []repoRoleActions
		UserID          int
		UserGroupRels   *userGroupRels
		ActionStr       string
		RepoID          int
		RepogroupRels   []*dblogic.RepogroupRel
		AssignedRoles   []*assignedRole
		DateTime        string
	}

	authzTemplStr := `
{{range .RepoRoleActions}}repo_role_actions("{{NamespaceRole .RoleName}}", [{{BuildRoleActions .RoleAllowedActions}}]);
{{end}}

operation("{{NamespaceAction .ActionStr}}", "{{NamespaceRepo .RepoID}}");
time({{.DateTime}});

repo($repoid) <-
  operation($action, $repoid);

{{with .UserGroupRels}}{{range .UserInGroups}}usergroup("{{NamespaceUG .UsergroupId}}", "{{NamespaceUser .UserId}}");
{{end}}{{range .UgsInUgs}}usergroup("{{NamespaceUG .ParentUsergroupId}}", "{{NamespaceUG .ChildUsergroupId}}");
{{end}}{{end}}

{{range .RepogroupRels}}repogroup("{{NamespaceRG .RepogroupId}}", "{{NamespaceRepo .RepoId}}");
{{end}}

{{range .AssignedRoles}}role("{{.UserNamespaced}}", "{{.RepoNamespaced}}", "{{.Role}}");
{{end}}

user_authority($member, $member) <-
  user($member);
user_authority($member, $group) <-
  usergroup($group, $member), 
  $member.starts_with("userid:");
user_authority($member, $subgroup) <-
  usergroup($group, $subgroup),
  $subgroup.starts_with("usergroupid:"),
  user_authority($member, $group);

repo_authority($member, $member) <-
  repo($member);
repo_authority($member, $group) <-
  repogroup($group, $member);

req_role($role, $action) <-
  operation($action, $repo),
  repo_role_actions($role, $permissions), $permissions.contains($action);

allow if
  user($user),
  operation($action, $repo),
  req_role($role, $action),
  user_authority($user, $userOrgroup),
  repo_authority($repo, $repoOrgroup),
  role($userOrGroup, $repoOrGroup, $role);

`
	tmpl := template.Must(template.New("DatalogAuthZ").
		Funcs(template.FuncMap{
			"BuildRoleActions": buildRoleActions,
			"NamespaceRole":    namespaceRole,
			"NamespaceUser":    namespaceUser,
			"NamespaceUG":      namespaceUG,
			"NamespaceRepo":    namespaceRepo,
			"NamespaceAction":  namespaceAction,
			"NamespaceRG":      namespaceRG,
		},
		).Parse(authzTemplStr))

	authzDetailsInst := authzDetails{
		// Describe role -> action logic. This is stored in code since it
		// describes logical operations. It _could_ be in a database
		// if more flexibility around defining roles was desired.
		RepoRoleActions: []repoRoleActions{
			repoRoleActions{
				RoleName: ownerRoleStr,
				RoleAllowedActions: []string{
					membershipStr,
					writeStr,
					readStr,
				},
			},
			repoRoleActions{
				RoleName: writerRoleStr,
				RoleAllowedActions: []string{
					writeStr,
					readStr,
				},
			},
			repoRoleActions{
				RoleName: readerRoleStr,
				RoleAllowedActions: []string{
					readStr,
				},
			},
		},
		UserID: reqDetails.UserId,
		UserGroupRels: &userGroupRels{
			UserInGroups: []*userInGroup{},
			UgsInUgs:     []*ugInUg{},
		},
		RepoID:        reqDetails.RepoId,
		RepogroupRels: reqDetails.RepogroupRels,
		AssignedRoles: []*assignedRole{},
		DateTime:      time.Now().UTC().Format(time.RFC3339),
	}

	switch operation {
	case Membership:
		authzDetailsInst.ActionStr = membershipStr
	case Read:
		authzDetailsInst.ActionStr = readStr
	case Write:
		authzDetailsInst.ActionStr = writeStr
	default:
		log.Fatalf("Unknown operation: %s", operation)
	}

	// Why not just use the dblogic UserRelationships directly? This
	// makes it easier to change it over time as needed in the template.
	// (I later give up for RepogroupRels)
	userGroupRelsPtr := reqDetails.UsergroupRelationships
	for _, userInGroupDB := range userGroupRelsPtr.UserInGroups {
		userInGroupTmpl := &userInGroup{
			UsergroupId: userInGroupDB.UsergroupId,
			UserId:      userInGroupDB.UserId,
		}
		authzDetailsInst.UserGroupRels.UserInGroups = append(
			authzDetailsInst.UserGroupRels.UserInGroups,
			userInGroupTmpl)
	}
	for _, ugsInUgsDB := range userGroupRelsPtr.UserGroupInGroups {
		ugInUgTmpl := &ugInUg{
			ParentUsergroupId: ugsInUgsDB.ParentUsergroupId,
			ChildUsergroupId:  ugsInUgsDB.ChildUsergroupId,
		}
		authzDetailsInst.UserGroupRels.UgsInUgs = append(
			authzDetailsInst.UserGroupRels.UgsInUgs,
			ugInUgTmpl)
	}

	// For better or worse, avoid using templates and build out
	// the role($userOrGroup, $repoOrGroup, $role) facts here
	for _, dbAssignRole := range reqDetails.AssignedRoles {
		userOrGroup := ""
		switch dbAssignRole.UserOrGroup {
		case dblogic.UserUGR:
			userOrGroup = namespaceUser(dbAssignRole.UserOrGroupID)
		case dblogic.UsergroupUGR:
			userOrGroup = namespaceUG(dbAssignRole.UserOrGroupID)
		default:
			log.Fatalf("Failed to match UserOrGroup in role assignment: %d", dbAssignRole.UserOrGroup)
		}
		repoOrGroup := ""
		switch dbAssignRole.RepoOrGroup {
		case dblogic.RepoUGR:
			repoOrGroup = namespaceRepo(dbAssignRole.RepoOrGroupID)
		case dblogic.RepogroupUGR:
			repoOrGroup = namespaceRG(dbAssignRole.RepoOrGroupID)
		default:
			log.Fatalf("Failed to match RepoOrGroup in role assignment: %d", dbAssignRole.RepoOrGroup)
		}
		roleName := ""
		switch dbAssignRole.RepoRole {
		case dblogic.OwnerRole:
			roleName = namespaceRole(ownerRoleStr)
		case dblogic.ReaderRole:
			roleName = namespaceRole(readerRoleStr)
		case dblogic.WriterRole:
			roleName = namespaceRole(writerRoleStr)
		default:
			log.Fatalf("Failed to match RepoRole in role assignment: %d", dbAssignRole.RepoRole)
		}

		assignedRoleMapping := &assignedRole{
			Role:           roleName,
			UserNamespaced: userOrGroup,
			RepoNamespaced: repoOrGroup,
		}
		authzDetailsInst.AssignedRoles = append(
			authzDetailsInst.AssignedRoles,
			assignedRoleMapping,
		)
	}

	buffer := &bytes.Buffer{}
	if err := tmpl.Execute(buffer, authzDetailsInst); err != nil {
		return false, fmt.Errorf("error executing template: %w", err)
	}

	log.Printf("Biscuit authorizer is:\n%s\n== END AUTHORIZER ==", buffer.String())

	authorizer, err := token.Authorizer(publicRoot)
	if err != nil {
		return false, fmt.Errorf("error when verifying token and creating authorizer: %w", err.Error())
	}
	authorizerContents, err := parser.FromStringAuthorizer(buffer.String())
	if err != nil {
		return false, fmt.Errorf("error when parsing authorizer: %w", err)
	}
	authorizer.AddAuthorizer(authorizerContents)

	err = authorizer.Authorize()
	log.Printf("Biscuit World (post auth) is:\n%s\n== END POST AUTH WORLD ==", authorizer.PrintWorld())
	if err != nil {
		return false, fmt.Errorf("error in Authorize: %w", err.Error())
	}

	return true, nil
}

// AttenuateBiscuit attenuates a biscuit with the provided set of checks
func AttenuateBiscuit(biscuitToken *biscuit.Biscuit, blockTxt string) (*biscuit.Biscuit, error) {
	blockBuilder := biscuitToken.CreateBlock()
	check, err := parser.FromStringCheck(blockTxt)
	if err != nil {
		return nil, fmt.Errorf("error when parsing repo attenuation: %w", err.Error())
	}
	err = blockBuilder.AddCheck(check)
	if err != nil {
		return nil, fmt.Errorf("error when adding check to block: %w", err.Error())
	}
	biscuitToken, err = biscuitToken.Append(rand.Reader, blockBuilder.Build())
	if err != nil {
		return nil, fmt.Errorf("error when appending new block to token: %w", err.Error())
	}

	return biscuitToken, nil
}
