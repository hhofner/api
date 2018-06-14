package swagger

import "git.kolaente.de/konrad/list/models"

// not actually a response, just a hack to get go-swagger to include definitions
// of the various XYZOption structs

// parameterBodies
// swagger:response parameterBodies
type swaggerParameterBodies struct {
	// in:body
	UserLogin models.UserLogin

	// in:body
	ApiUserPassword models.ApiUserPassword

	// in:body
	List models.List

	// in:body
	ListItem models.ListItem
}