package models

// CreateOrUpdateList updates a list or creates it if it doesn't exist
func CreateOrUpdateList(list *List) (err error) {

	// Check we have at least a title
	if list.Title == "" {
		return ErrListTitleCannotBeEmpty{}
	}

	// Check if the namespace exists
	if list.NamespaceID != 0 {
		_, err = GetNamespaceByID(list.NamespaceID)
		if err != nil {
			return err
		}
	}

	if list.ID == 0 {
		_, err = x.Insert(list)
	} else {
		_, err = x.ID(list.ID).Update(list)
	}

	if err != nil {
		return
	}

	err = list.ReadOne()
	return

}

// Update implements the update method of CRUDable
func (l *List) Update() (err error) {
	// Check if it exists
	if err = l.GetSimpleByID(); err != nil {
		return
	}

	return CreateOrUpdateList(l)
}

// Create implements the create method of CRUDable
func (l *List) Create(doer *User) (err error) {
	// Check rights
	u, err := GetUserByID(doer.ID)
	if err != nil {
		return
	}

	l.OwnerID = u.ID
	l.Owner.ID = u.ID
	l.ID = 0 // Otherwise only the first time a new list would be created

	return CreateOrUpdateList(l)
}