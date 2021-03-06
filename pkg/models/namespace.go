// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-2020 Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"sort"
	"time"

	"code.vikunja.io/api/pkg/metrics"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/web"
	"github.com/imdario/mergo"
	"xorm.io/builder"
)

// Namespace holds informations about a namespace
type Namespace struct {
	// The unique, numeric id of this namespace.
	ID int64 `xorm:"int(11) autoincr not null unique pk" json:"id" param:"namespace"`
	// The name of this namespace.
	Title string `xorm:"varchar(250) not null" json:"title" valid:"required,runelength(1|250)" minLength:"1" maxLength:"250"`
	// The description of the namespace
	Description string `xorm:"longtext null" json:"description"`
	OwnerID     int64  `xorm:"int(11) not null INDEX" json:"-"`

	// The hex color of this namespace
	HexColor string `xorm:"varchar(6) null" json:"hex_color" valid:"runelength(0|6)" maxLength:"6"`

	// Whether or not a namespace is archived.
	IsArchived bool `xorm:"not null default false" json:"is_archived" query:"is_archived"`

	// The user who owns this namespace
	Owner *user.User `xorm:"-" json:"owner" valid:"-"`

	// A timestamp when this namespace was created. You cannot change this value.
	Created time.Time `xorm:"created not null" json:"created"`
	// A timestamp when this namespace was last updated. You cannot change this value.
	Updated time.Time `xorm:"updated not null" json:"updated"`

	web.CRUDable `xorm:"-" json:"-"`
	web.Rights   `xorm:"-" json:"-"`
}

// SharedListsPseudoNamespace is a pseudo namespace used to hold shared lists
var SharedListsPseudoNamespace = Namespace{
	ID:          -1,
	Title:       "Shared Lists",
	Description: "Lists of other users shared with you via teams or directly.",
	Created:     time.Now(),
	Updated:     time.Now(),
}

// FavoritesPseudoNamespace is a pseudo namespace used to hold favorited lists and tasks
var FavoritesPseudoNamespace = Namespace{
	ID:          -2,
	Title:       "Favorites",
	Description: "Favorite lists and tasks.",
	Created:     time.Now(),
	Updated:     time.Now(),
}

// SavedFiltersPseudoNamespace is a pseudo namespace used to hold saved filters
var SavedFiltersPseudoNamespace = Namespace{
	ID:          -3,
	Title:       "Filters",
	Description: "Saved filters.",
	Created:     time.Now(),
	Updated:     time.Now(),
}

// TableName makes beautiful table names
func (Namespace) TableName() string {
	return "namespaces"
}

// GetSimpleByID gets a namespace without things like the owner, it more or less only checks if it exists.
func (n *Namespace) GetSimpleByID() (err error) {
	if n.ID == 0 {
		return ErrNamespaceDoesNotExist{ID: n.ID}
	}

	// Get the namesapce with shared lists
	if n.ID == -1 {
		*n = SharedListsPseudoNamespace
		return
	}

	if n.ID == FavoritesPseudoNamespace.ID {
		*n = FavoritesPseudoNamespace
		return
	}

	namespaceFromDB := &Namespace{}
	exists, err := x.Where("id = ?", n.ID).Get(namespaceFromDB)
	if err != nil {
		return
	}
	if !exists {
		return ErrNamespaceDoesNotExist{ID: n.ID}
	}
	// We don't want to override the provided user struct because this would break updating, so we have to merge it
	if err := mergo.Merge(namespaceFromDB, n, mergo.WithOverride); err != nil {
		return err
	}
	*n = *namespaceFromDB

	return
}

// GetNamespaceByID returns a namespace object by its ID
func GetNamespaceByID(id int64) (namespace Namespace, err error) {
	namespace = Namespace{ID: id}
	err = namespace.GetSimpleByID()
	if err != nil {
		return
	}

	// Get the namespace Owner
	namespace.Owner, err = user.GetUserByID(namespace.OwnerID)
	return
}

// CheckIsArchived returns an ErrNamespaceIsArchived if the namepace is archived.
func (n *Namespace) CheckIsArchived() error {
	exists, err := x.
		Where("id = ? AND is_archived = true", n.ID).
		Exist(&Namespace{})
	if err != nil {
		return err
	}
	if exists {
		return ErrNamespaceIsArchived{NamespaceID: n.ID}
	}
	return nil
}

// ReadOne gets one namespace
// @Summary Gets one namespace
// @Description Returns a namespace by its ID.
// @tags namespace
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param id path int true "Namespace ID"
// @Success 200 {object} models.Namespace "The Namespace"
// @Failure 403 {object} web.HTTPError "The user does not have access to that namespace."
// @Failure 500 {object} models.Message "Internal error"
// @Router /namespaces/{id} [get]
func (n *Namespace) ReadOne() (err error) {
	*n, err = GetNamespaceByID(n.ID)
	return
}

// NamespaceWithLists represents a namespace with list meta informations
type NamespaceWithLists struct {
	Namespace `xorm:"extends"`
	Lists     []*List `xorm:"-" json:"lists"`
}

// ReadAll gets all namespaces a user has access to
// @Summary Get all namespaces a user has access to
// @Description Returns all namespaces a user has access to.
// @tags namespace
// @Accept json
// @Produce json
// @Param page query int false "The page number. Used for pagination. If not provided, the first page of results is returned."
// @Param per_page query int false "The maximum number of items per page. Note this parameter is limited by the configured maximum of items per page."
// @Param s query string false "Search namespaces by name."
// @Param is_archived query bool false "If true, also returns all archived namespaces."
// @Security JWTKeyAuth
// @Success 200 {array} models.NamespaceWithLists "The Namespaces."
// @Failure 500 {object} models.Message "Internal error"
// @Router /namespaces [get]
func (n *Namespace) ReadAll(a web.Auth, search string, page int, perPage int) (result interface{}, resultCount int, numberOfTotalItems int64, err error) {
	if _, is := a.(*LinkSharing); is {
		return nil, 0, 0, ErrGenericForbidden{}
	}

	// This map will hold all namespaces and their lists. The key is usually the id of the namespace.
	// We're using a map here because it makes a few things like adding lists or removing pseudo namespaces easier.
	namespaces := make(map[int64]*NamespaceWithLists)

	//////////////////////////////
	// Lists with their namespaces

	doer, err := user.GetFromAuth(a)
	if err != nil {
		return nil, 0, 0, err
	}

	// Adding a 1=1 condition by default here because xorm always needs a condition and cannot handle nil conditions
	var isArchivedCond builder.Cond = builder.Eq{"1": 1}
	if !n.IsArchived {
		isArchivedCond = builder.And(
			builder.Eq{"namespaces.is_archived": false},
		)
	}

	limit, start := getLimitFromPageIndex(page, perPage)
	query := x.Select("namespaces.*").
		Table("namespaces").
		Join("LEFT", "team_namespaces", "namespaces.id = team_namespaces.namespace_id").
		Join("LEFT", "team_members", "team_members.team_id = team_namespaces.team_id").
		Join("LEFT", "users_namespace", "users_namespace.namespace_id = namespaces.id").
		Where("team_members.user_id = ?", doer.ID).
		Or("namespaces.owner_id = ?", doer.ID).
		Or("users_namespace.user_id = ?", doer.ID).
		GroupBy("namespaces.id").
		Where("namespaces.title LIKE ?", "%"+search+"%").
		Where(isArchivedCond)
	if limit > 0 {
		query = query.Limit(limit, start)
	}
	err = query.Find(&namespaces)
	if err != nil {
		return nil, 0, 0, err
	}

	// Make a list of namespace ids
	var namespaceids []int64
	var userIDs []int64
	for _, nsp := range namespaces {
		namespaceids = append(namespaceids, nsp.ID)
		userIDs = append(userIDs, nsp.OwnerID)
	}

	// Get all owners
	userMap := make(map[int64]*user.User)
	err = x.In("id", userIDs).Find(&userMap)
	if err != nil {
		return nil, 0, 0, err
	}

	// Get all lists
	lists := []*List{}
	listQuery := x.
		In("namespace_id", namespaceids)

	if !n.IsArchived {
		listQuery.And("is_archived = false")
	}
	err = listQuery.Find(&lists)
	if err != nil {
		return nil, 0, 0, err
	}

	numberOfTotalItems, err = x.
		Table("namespaces").
		Join("LEFT", "team_namespaces", "namespaces.id = team_namespaces.namespace_id").
		Join("LEFT", "team_members", "team_members.team_id = team_namespaces.team_id").
		Join("LEFT", "users_namespace", "users_namespace.namespace_id = namespaces.id").
		Where("team_members.user_id = ?", doer.ID).
		Or("namespaces.owner_id = ?", doer.ID).
		Or("users_namespace.user_id = ?", doer.ID).
		And("namespaces.is_archived = false").
		GroupBy("namespaces.id").
		Where("namespaces.title LIKE ?", "%"+search+"%").
		Count(&NamespaceWithLists{})
	if err != nil {
		return nil, 0, 0, err
	}

	///////////////
	// Shared Lists

	// Create our pseudo namespace to hold the shared lists
	sharedListsPseudonamespace := SharedListsPseudoNamespace
	sharedListsPseudonamespace.Owner = doer
	namespaces[sharedListsPseudonamespace.ID] = &NamespaceWithLists{
		sharedListsPseudonamespace,
		[]*List{},
	}

	// Get all lists individually shared with our user (not via a namespace)
	individualLists := []*List{}
	iListQuery := x.Select("l.*").
		Table("list").
		Alias("l").
		Join("LEFT", []string{"team_list", "tl"}, "l.id = tl.list_id").
		Join("LEFT", []string{"team_members", "tm"}, "tm.team_id = tl.team_id").
		Join("LEFT", []string{"users_list", "ul"}, "ul.list_id = l.id").
		Where("tm.user_id = ?", doer.ID).
		Or("ul.user_id = ?", doer.ID).
		GroupBy("l.id")
	if !n.IsArchived {
		iListQuery.And("l.is_archived = false")
	}
	err = iListQuery.Find(&individualLists)
	if err != nil {
		return nil, 0, 0, err
	}

	// Make the namespace -1 so we now later which one it was
	// + Append it to all lists we already have
	for _, l := range individualLists {
		l.NamespaceID = -1
		lists = append(lists, l)
	}

	// Remove the sharedListsPseudonamespace if we don't have any shared lists
	if len(individualLists) == 0 {
		delete(namespaces, sharedListsPseudonamespace.ID)
	}

	// More details for the lists
	err = AddListDetails(lists)
	if err != nil {
		return nil, 0, 0, err
	}

	/////////////////
	// Favorite lists

	// Create our pseudo namespace with favorite lists
	pseudoFavoriteNamespace := FavoritesPseudoNamespace
	pseudoFavoriteNamespace.Owner = doer
	namespaces[pseudoFavoriteNamespace.ID] = &NamespaceWithLists{
		Namespace: pseudoFavoriteNamespace,
		Lists:     []*List{{}},
	}
	*namespaces[pseudoFavoriteNamespace.ID].Lists[0] = FavoritesPseudoList // Copying the list to be able to modify it later

	for _, list := range lists {
		if list.IsFavorite {
			namespaces[pseudoFavoriteNamespace.ID].Lists = append(namespaces[pseudoFavoriteNamespace.ID].Lists, list)
		}
		namespaces[list.NamespaceID].Lists = append(namespaces[list.NamespaceID].Lists, list)
	}

	// Check if we have any favorites or favorited lists and remove the favorites namespace from the list if not
	var favoriteCount int64
	favoriteCount, err = x.
		Join("INNER", "list", "tasks.list_id = list.id").
		Join("INNER", "namespaces", "list.namespace_id = namespaces.id").
		Where(builder.And(builder.Eq{"tasks.is_favorite": true}, builder.In("namespaces.id", namespaceids))).
		Count(&Task{})
	if err != nil {
		return nil, 0, 0, err
	}

	// If we don't have any favorites in the favorites pseudo list, remove that pseudo list from the namespace
	if favoriteCount == 0 {
		for in, l := range namespaces[pseudoFavoriteNamespace.ID].Lists {
			if l.ID == FavoritesPseudoList.ID {
				namespaces[pseudoFavoriteNamespace.ID].Lists = append(namespaces[pseudoFavoriteNamespace.ID].Lists[:in], namespaces[pseudoFavoriteNamespace.ID].Lists[in+1:]...)
				break
			}
		}
	}

	// If we don't have any favorites in the namespace, remove it
	if len(namespaces[pseudoFavoriteNamespace.ID].Lists) == 0 {
		delete(namespaces, pseudoFavoriteNamespace.ID)
	}

	/////////////////
	// Saved Filters

	savedFilters, err := getSavedFiltersForUser(a)
	if err != nil {
		return nil, 0, 0, err
	}

	if len(savedFilters) > 0 {
		savedFiltersPseudoNamespace := SavedFiltersPseudoNamespace
		savedFiltersPseudoNamespace.Owner = doer
		namespaces[savedFiltersPseudoNamespace.ID] = &NamespaceWithLists{
			Namespace: savedFiltersPseudoNamespace,
			Lists:     make([]*List, 0, len(savedFilters)),
		}

		for _, filter := range savedFilters {
			namespaces[savedFiltersPseudoNamespace.ID].Lists = append(namespaces[savedFiltersPseudoNamespace.ID].Lists, &List{
				ID:          getListIDFromSavedFilterID(filter.ID),
				Title:       filter.Title,
				Description: filter.Description,
				Created:     filter.Created,
				Updated:     filter.Updated,
				Owner:       doer,
			})
		}
	}

	//////////////////////
	// Put it all together (and sort it)
	all := make([]*NamespaceWithLists, 0, len(namespaces))
	for _, n := range namespaces {
		n.Owner = userMap[n.OwnerID]
		all = append(all, n)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})

	return all, len(all), numberOfTotalItems, nil
}

// Create implements the creation method via the interface
// @Summary Creates a new namespace
// @Description Creates a new namespace.
// @tags namespace
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param namespace body models.Namespace true "The namespace you want to create."
// @Success 200 {object} models.Namespace "The created namespace."
// @Failure 400 {object} web.HTTPError "Invalid namespace object provided."
// @Failure 403 {object} web.HTTPError "The user does not have access to the namespace"
// @Failure 500 {object} models.Message "Internal error"
// @Router /namespaces [put]
func (n *Namespace) Create(a web.Auth) (err error) {
	// Check if we have at least a name
	if n.Title == "" {
		return ErrNamespaceNameCannotBeEmpty{NamespaceID: 0, UserID: a.GetID()}
	}
	n.ID = 0 // This would otherwise prevent the creation of new lists after one was created

	// Check if the User exists
	n.Owner, err = user.GetUserByID(a.GetID())
	if err != nil {
		return
	}
	n.OwnerID = n.Owner.ID

	// Insert
	if _, err = x.Insert(n); err != nil {
		return err
	}

	metrics.UpdateCount(1, metrics.NamespaceCountKey)
	return
}

// Delete deletes a namespace
// @Summary Deletes a namespace
// @Description Delets a namespace
// @tags namespace
// @Produce json
// @Security JWTKeyAuth
// @Param id path int true "Namespace ID"
// @Success 200 {object} models.Message "The namespace was successfully deleted."
// @Failure 400 {object} web.HTTPError "Invalid namespace object provided."
// @Failure 403 {object} web.HTTPError "The user does not have access to the namespace"
// @Failure 500 {object} models.Message "Internal error"
// @Router /namespaces/{id} [delete]
func (n *Namespace) Delete() (err error) {

	// Check if the namespace exists
	_, err = GetNamespaceByID(n.ID)
	if err != nil {
		return
	}

	// Delete the namespace
	_, err = x.ID(n.ID).Delete(&Namespace{})
	if err != nil {
		return
	}

	// Delete all lists with their tasks
	lists, err := GetListsByNamespaceID(n.ID, &user.User{})
	if err != nil {
		return
	}
	var listIDs []int64
	// We need to do that for here because we need the list ids to delete two times:
	// 1) to delete the lists itself
	// 2) to delete the list tasks
	for _, l := range lists {
		listIDs = append(listIDs, l.ID)
	}

	// Delete tasks
	_, err = x.In("list_id", listIDs).Delete(&Task{})
	if err != nil {
		return
	}

	// Delete the lists
	_, err = x.In("id", listIDs).Delete(&List{})
	if err != nil {
		return
	}

	metrics.UpdateCount(-1, metrics.NamespaceCountKey)

	return
}

// Update implements the update method via the interface
// @Summary Updates a namespace
// @Description Updates a namespace.
// @tags namespace
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param id path int true "Namespace ID"
// @Param namespace body models.Namespace true "The namespace with updated values you want to update."
// @Success 200 {object} models.Namespace "The updated namespace."
// @Failure 400 {object} web.HTTPError "Invalid namespace object provided."
// @Failure 403 {object} web.HTTPError "The user does not have access to the namespace"
// @Failure 500 {object} models.Message "Internal error"
// @Router /namespace/{id} [post]
func (n *Namespace) Update() (err error) {
	// Check if we have at least a name
	if n.Title == "" {
		return ErrNamespaceNameCannotBeEmpty{NamespaceID: n.ID}
	}

	// Check if the namespace exists
	currentNamespace, err := GetNamespaceByID(n.ID)
	if err != nil {
		return
	}

	// Check if the namespace is archived and the update is not un-archiving it
	if currentNamespace.IsArchived && n.IsArchived {
		return ErrNamespaceIsArchived{NamespaceID: n.ID}
	}

	// Check if the (new) owner exists
	if n.Owner != nil {
		n.OwnerID = n.Owner.ID
		if currentNamespace.OwnerID != n.OwnerID {
			n.Owner, err = user.GetUserByID(n.OwnerID)
			if err != nil {
				return
			}
		}
	}

	// We need to specify the cols we want to update here to be able to un-archive lists
	colsToUpdate := []string{
		"title",
		"is_archived",
		"hex_color",
	}
	if n.Description != "" {
		colsToUpdate = append(colsToUpdate, "description")
	}

	// Do the actual update
	_, err = x.
		ID(currentNamespace.ID).
		Cols(colsToUpdate...).
		Update(n)
	return
}
