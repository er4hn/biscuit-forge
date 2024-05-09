--
-- File generated with SQLiteStudio v3.4.4
--
-- Text encoding used: System
--
PRAGMA foreign_keys = off;
BEGIN TRANSACTION;

-- Table: repo_roles_enum
CREATE TABLE IF NOT EXISTS repo_roles_enum (
    id       INTEGER PRIMARY KEY
                     UNIQUE
                     NOT NULL,
    rolename TEXT    UNIQUE
                     NOT NULL
);

INSERT INTO repo_roles_enum (
                                id,
                                rolename
                            )
                            VALUES (
                                1,
                                'reader'
                            );

INSERT INTO repo_roles_enum (
                                id,
                                rolename
                            )
                            VALUES (
                                2,
                                'writer'
                            );

INSERT INTO repo_roles_enum (
                                id,
                                rolename
                            )
                            VALUES (
                                3,
                                'owner'
                            );


-- Table: Repo_Roles_membership_UserGroups
CREATE TABLE IF NOT EXISTS Repo_Roles_membership_UserGroups (
    id           INTEGER PRIMARY KEY
                         UNIQUE
                         NOT NULL,
    repo_id      INTEGER REFERENCES Repos (id) ON DELETE CASCADE
                         NOT NULL,
    usergroup_id INTEGER NOT NULL
                         REFERENCES UserGroups (id) ON DELETE CASCADE,
    repo_role    INTEGER REFERENCES repo_roles_enum (id) 
                         NOT NULL
);


-- Table: Repo_Roles_membership_Users
CREATE TABLE IF NOT EXISTS Repo_Roles_membership_Users (
    id        INTEGER PRIMARY KEY
                      UNIQUE
                      NOT NULL,
    repo_id   INTEGER REFERENCES Repos (id) ON DELETE CASCADE
                      NOT NULL,
    user_id   INTEGER NOT NULL
                      REFERENCES Users (id) ON DELETE CASCADE,
    repo_role INTEGER REFERENCES repo_roles_enum (id) 
                      NOT NULL
);

INSERT INTO Repo_Roles_membership_Users (
                                            id,
                                            repo_id,
                                            user_id,
                                            repo_role
                                        )
                                        VALUES (
                                            1,
                                            3,
                                            1,
                                            3
                                        );

INSERT INTO Repo_Roles_membership_Users (
                                            id,
                                            repo_id,
                                            user_id,
                                            repo_role
                                        )
                                        VALUES (
                                            2,
                                            3,
                                            2,
                                            1
                                        );


-- Table: RepoGroup_membership
CREATE TABLE IF NOT EXISTS RepoGroup_membership (
    id           INTEGER PRIMARY KEY
                         UNIQUE
                         NOT NULL,
    repogroup_id INTEGER REFERENCES RepoGroups (id) ON DELETE CASCADE
                         NOT NULL,
    repo_id      INTEGER REFERENCES Repos (id) ON DELETE CASCADE
                         NOT NULL
);

INSERT INTO RepoGroup_membership (
                                     id,
                                     repogroup_id,
                                     repo_id
                                 )
                                 VALUES (
                                     1,
                                     1,
                                     2
                                 );

INSERT INTO RepoGroup_membership (
                                     id,
                                     repogroup_id,
                                     repo_id
                                 )
                                 VALUES (
                                     2,
                                     1,
                                     3
                                 );


-- Table: repogroup_roles_enum
CREATE TABLE IF NOT EXISTS repogroup_roles_enum (
    id       INTEGER PRIMARY KEY
                     UNIQUE
                     NOT NULL,
    rolename TEXT    UNIQUE
                     NOT NULL
);

INSERT INTO repogroup_roles_enum (
                                     id,
                                     rolename
                                 )
                                 VALUES (
                                     1,
                                     'reader'
                                 );

INSERT INTO repogroup_roles_enum (
                                     id,
                                     rolename
                                 )
                                 VALUES (
                                     2,
                                     'writer'
                                 );


-- Table: RepoGroup_Roles_membership_Usergroup
CREATE TABLE IF NOT EXISTS RepoGroup_Roles_membership_Usergroup (
    id             INTEGER PRIMARY KEY
                           UNIQUE
                           NOT NULL,
    repogroup_id   INTEGER REFERENCES RepoGroups (id) ON DELETE CASCADE
                           NOT NULL,
    usergroup_id   INTEGER NOT NULL
                           REFERENCES UserGroups (id) ON DELETE CASCADE,
    repogroup_role INTEGER REFERENCES repogroup_roles_enum (id) 
                           NOT NULL
);

INSERT INTO RepoGroup_Roles_membership_Usergroup (
                                                     id,
                                                     repogroup_id,
                                                     usergroup_id,
                                                     repogroup_role
                                                 )
                                                 VALUES (
                                                     1,
                                                     1,
                                                     1,
                                                     2
                                                 );


-- Table: RepoGroup_Roles_membership_Users
CREATE TABLE IF NOT EXISTS RepoGroup_Roles_membership_Users (
    id             INTEGER PRIMARY KEY
                           UNIQUE
                           NOT NULL,
    repogroup_id   INTEGER REFERENCES RepoGroups (id) ON DELETE CASCADE
                           NOT NULL,
    user_id        INTEGER NOT NULL
                           REFERENCES Users (id) ON DELETE CASCADE,
    repogroup_role INTEGER REFERENCES repogroup_roles_enum (id) 
                           NOT NULL
);


-- Table: RepoGroups
CREATE TABLE IF NOT EXISTS RepoGroups (
    id        INTEGER PRIMARY KEY
                      UNIQUE
                      NOT NULL,
    groupname TEXT    UNIQUE
                      NOT NULL
);

INSERT INTO RepoGroups (
                           id,
                           groupname
                       )
                       VALUES (
                           1,
                           'Foo'
                       );


-- Table: Repos
CREATE TABLE IF NOT EXISTS Repos (
    id       INTEGER PRIMARY KEY
                     UNIQUE
                     NOT NULL,
    reponame TEXT    UNIQUE
                     NOT NULL
);

INSERT INTO Repos (
                      id,
                      reponame
                  )
                  VALUES (
                      1,
                      'Alpha'
                  );

INSERT INTO Repos (
                      id,
                      reponame
                  )
                  VALUES (
                      2,
                      'Bravo'
                  );

INSERT INTO Repos (
                      id,
                      reponame
                  )
                  VALUES (
                      3,
                      'Charlie'
                  );


-- Table: UserGroup_membership_usergroups
CREATE TABLE IF NOT EXISTS UserGroup_membership_usergroups (
    id                 INTEGER PRIMARY KEY
                               UNIQUE
                               NOT NULL,
    usergroup_id       INTEGER REFERENCES UserGroups (id) ON DELETE CASCADE
                               NOT NULL,
    child_usergroup_id INTEGER REFERENCES UserGroups (id) ON DELETE CASCADE
                               NOT NULL
);

INSERT INTO UserGroup_membership_usergroups (
                                                id,
                                                usergroup_id,
                                                child_usergroup_id
                                            )
                                            VALUES (
                                                1,
                                                1,
                                                2
                                            );

INSERT INTO UserGroup_membership_usergroups (
                                                id,
                                                usergroup_id,
                                                child_usergroup_id
                                            )
                                            VALUES (
                                                2,
                                                2,
                                                3
                                            );


-- Table: UserGroup_membership_users
CREATE TABLE IF NOT EXISTS UserGroup_membership_users (
    id           INTEGER PRIMARY KEY
                         UNIQUE
                         NOT NULL,
    usergroup_id INTEGER REFERENCES UserGroups (id) ON DELETE CASCADE
                         NOT NULL,
    user_id      INTEGER REFERENCES Users (id) ON DELETE CASCADE
                         NOT NULL
);

INSERT INTO UserGroup_membership_users (
                                           id,
                                           usergroup_id,
                                           user_id
                                       )
                                       VALUES (
                                           1,
                                           1,
                                           4
                                       );

INSERT INTO UserGroup_membership_users (
                                           id,
                                           usergroup_id,
                                           user_id
                                       )
                                       VALUES (
                                           2,
                                           1,
                                           3
                                       );

INSERT INTO UserGroup_membership_users (
                                           id,
                                           usergroup_id,
                                           user_id
                                       )
                                       VALUES (
                                           3,
                                           3,
                                           5
                                       );


-- Table: UserGroups
CREATE TABLE IF NOT EXISTS UserGroups (
    id        INTEGER PRIMARY KEY
                      NOT NULL
                      UNIQUE,
    groupname TEXT    UNIQUE
                      NOT NULL
);

INSERT INTO UserGroups (
                           id,
                           groupname
                       )
                       VALUES (
                           1,
                           'FooOps'
                       );

INSERT INTO UserGroups (
                           id,
                           groupname
                       )
                       VALUES (
                           2,
                           'BarOps'
                       );

INSERT INTO UserGroups (
                           id,
                           groupname
                       )
                       VALUES (
                           3,
                           'BazOps'
                       );


-- Table: Users
CREATE TABLE IF NOT EXISTS Users (
    id       INTEGER PRIMARY KEY
                     NOT NULL
                     UNIQUE,
    username TEXT    UNIQUE
                     NOT NULL
);

INSERT INTO Users (
                      id,
                      username
                  )
                  VALUES (
                      1,
                      'Olivia'
                  );

INSERT INTO Users (
                      id,
                      username
                  )
                  VALUES (
                      2,
                      'Noah'
                  );

INSERT INTO Users (
                      id,
                      username
                  )
                  VALUES (
                      3,
                      'Emma'
                  );

INSERT INTO Users (
                      id,
                      username
                  )
                  VALUES (
                      4,
                      'Liam'
                  );

INSERT INTO Users (
                      id,
                      username
                  )
                  VALUES (
                      5,
                      'Tony'
                  );


COMMIT TRANSACTION;
PRAGMA foreign_keys = on;
