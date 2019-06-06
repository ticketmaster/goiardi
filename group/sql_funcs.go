/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package group

// SQL goodies for groups

import (
	"database/sql"
	"errors"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/user"
	"github.com/lib/pq"
)

// Arrrgh, that's right. I need to look up the selecting an array aggregate with
// table join and so forth for groups

/**************

Phew, figured out the query to use to get groups & their members. Here it is
for reference:

-----
select name, organization_id, u.user_ids, c.client_ids, mg.group_ids FROM groups LEFT JOIN 
	(select gau.group_id AS ugid, array_agg(gau.user_id) AS user_ids FROM group_actor_users gau join groups g ON g.id = gau.group_id group by gau.group_id) u ON u.ugid = groups.id 
 LEFT JOIN 
	(select gac.group_id AS cgid, array_agg(gac.client_id) AS client_ids FROM group_actor_clients gac join groups g ON g.id = gac.group_id group by gac.group_id) c ON c.cgid = groups.id
 LEFT JOIN 
	(select gg.group_id AS ggid, array_agg(gg.member_group_id) AS group_ids FROM group_groups gg join groups g ON g.id = gg.group_id group by gg.group_id) mg ON mg.ggid = groups.id
WHERE groups.id = 1;
-----

It does, of course, need some cleaning up.

***************/

func checkForGroupSQL(dbhandle datastore.Dbhandle, org *organization.Organization, name string) (bool, error) {
	_, err := datastore.CheckForOne(dbhandle, "groups", name)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	return false, nil
}

func (g *Group) fillGroupFromSQL(row datastore.ResRow) error {
	var userIds []int64
	var clientIds []int64
	var groupIds []int64
	var orgId int64
	
	// arrrgh blargh, it looks like we may also need to create a special
	// type for getting the arrays of ints out of postgres.

	// eeesh, there isn't a whole lot we can fill in directly.
	err := row.Scan(&g.Name, &orgId, &userIds, &clientIds, &groupIds)
	if err != nil {
		return err
	}

	// TODO: check and make sure there's some way to prevent loops where
	// groups include each other as members.

	// fill in the actor and group slices with the appropriate objects
	// Will these need to be sorted? We'll see.
	// actorez := make([]actor.Actor, 0, len(userIds) + len(clientIds))

	groupez, err := GroupsByIdSQL(groupIds)
	if err != nil {
		return err
	}
	g.Groups = groupez

	userez, err := user.UsersByIdSQL(userIds)
	if err != nil {
		return err
	}

	clientez, err := client.ClientsByIdSQL(clientIds)
	if err != nil {
		return nil
	}

	actorez := make([]actor.Actor, len(userez) + len(clientez))
	// may need to do the explicit for range loop.
	actorez = append(actorez, userez...)
	actorez = append(actorez, clientez...)
}

func getGroupSQL(name string, org *organization.Organization) (*Group, error) {
	var sqlStatement string
	g := new(Group)

	if config.Config.UseMySQL {
		// MySQL will be rather more intricate than postgres, I'm
		// afraid. Leaving this here for now.
		sqlStatement = "SELECT name, organization_id FROM groups WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		// bleh, break this apart into multiple lines so there's some
		// small hope of reading and understanding it later.
		sqlStatement = `select name, organization_id, u.user_ids, c.client_ids, mg.group_ids FROM goiardi.groups g
		LEFT JOIN 
			(SELECT gau.group_id AS ugid, ARRAY_AGG(gau.user_id) AS user_ids FROM goiardi.group_actor_users gau JOIN goiardi.groups gs ON gs.id = gau.group_id group by gau.group_id) u ON u.ugid = groups.id 
		LEFT JOIN 
			(SELECT gac.group_id AS cgid, ARRAY_AGG(gac.client_id) AS client_ids FROM goiardi.group_actor_clients gac JOIN goiardi.groups gs ON gs.id = gac.group_id group by gac.group_id) c ON c.cgid = groups.id
		LEFT JOIN 
			(SELECT gg.group_id AS ggid, ARRAY_AGG(gg.member_group_id) AS group_ids FROM goiardi.group_groups gg JOIN goiardi.groups gs ON gs.id = gg.group_id group by gg.group_id) mg ON mg.ggid = groups.id
		WHERE organization_id = $1 AND name = $2`
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(org.GetId(), name);
	if err = g.fillGroupFromSQL(row); err != nil {
		return nil, err
	}
	return g, nil
}

func GroupsByIdSQL(ids []int64) ([]*Group, error) {
	if !config.UsingDB() {
		return nil, errors.New("GroupsByIdSQL only works if you're using a database storage backend.")
	}

	var groups []*User
	var sqlStatement string

	bind := make([]string, len(ids))
	intfIds := make([]interface{}, len(ids))

	if config.Config.UseMySQL {
		return nil, errors.New("Groups are not implemented with the MySQL backend yet, punting for now.")
	} else if config.Config.UsePostgreSQL {
		for i, d := range ids {
			bind[i] = fmt.Sprintf("$%d", i + 1)
			intfIds[i] = d
		}
		sqlStatement = fmt.Sprintf(`select name, organization_id, u.user_ids, c.client_ids, mg.group_ids FROM goiardi.groups g
		LEFT JOIN 
			(SELECT gau.group_id AS ugid, ARRAY_AGG(gau.user_id) AS user_ids FROM goiardi.group_actor_users gau JOIN goiardi.groups gs ON gs.id = gau.group_id group by gau.group_id) u ON u.ugid = groups.id 
		LEFT JOIN 
			(SELECT gac.group_id AS cgid, ARRAY_AGG(gac.client_id) AS client_ids FROM goiardi.group_actor_clients gac JOIN goiardi.groups gs ON gs.id = gac.group_id group by gac.group_id) c ON c.cgid = groups.id
		LEFT JOIN 
			(SELECT gg.group_id AS ggid, ARRAY_AGG(gg.member_group_id) AS group_ids FROM goiardi.group_groups gg JOIN goiardi.groups gs ON gs.id = gg.group_id group by gg.group_id) mg ON mg.ggid = groups.id
		WHERE id in (%s)`, strings.Join(bind, ", "))
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(intfIds...)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return users, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		mg := new(Group)
		err = mg.fillGroupFromSQL(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, mg)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}
