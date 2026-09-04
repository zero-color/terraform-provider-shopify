package shopify

import (
	"context"
)

type MetaobjectAccess struct {
	Admin      string `json:"admin,omitempty"`
	Storefront string `json:"storefront,omitempty"`
}

type MetaobjectDefinition struct {
	ID                string                       `json:"id"`
	Type              string                       `json:"type"`
	Name              string                       `json:"name"`
	Description       string                       `json:"description,omitempty"`
	DisplayNameKey    *string                      `json:"displayNameKey,omitempty"`
	FieldDefinitions  []*MetaobjectFieldDefinition `json:"fieldDefinitions"`
	HasThumbnailField bool                         `json:"hasThumbnailField"`
	Access            *MetaobjectAccess            `json:"access"`
}

type MetaobjectFieldDefinition struct {
	Key               string                           `json:"key"`
	Name              string                           `json:"name"`
	Description       string                           `json:"description,omitempty"`
	Type              *MetafieldDefinitionType         `json:"type"`
	Required          bool                             `json:"required"`
	HasThumbnailField bool                             `json:"hasThumbnailField"`
	Validations       []*MetafieldDefinitionValidation `json:"validations"`
}

type MetaobjectDefinitionCreateInput struct {
	Type             string                                  `json:"type"`
	Name             string                                  `json:"name"`
	Description      *string                                 `json:"description,omitempty"`
	DisplayNameKey   *string                                 `json:"displayNameKey,omitempty"`
	FieldDefinitions []*MetaobjectFieldDefinitionCreateInput `json:"fieldDefinitions"`
	Access           *MetaobjectAccess                       `json:"access,omitempty"`
}

type MetaobjectFieldDefinitionCreateInput struct {
	Key         string                           `json:"key"`
	Name        *string                          `json:"name,omitempty"`
	Description *string                          `json:"description,omitempty"`
	Type        string                           `json:"type"`
	Required    bool                             `json:"required"`
	Validations []*MetafieldDefinitionValidation `json:"validations"`
}

func (c *Client) CreateMetaobjectDefinition(ctx context.Context, input *MetaobjectDefinitionCreateInput) (*MetaobjectDefinition, error) {
	variables := map[string]interface{}{"definition": input}
	query := `
mutation CreateMetaobjectDefinition($definition: MetaobjectDefinitionCreateInput!) {
  metaobjectDefinitionCreate(definition: $definition) {
    metaobjectDefinition {
      id
      type
      name
      description
      displayNameKey
      fieldDefinitions {	
		key
		name
		description
		type {
		  category
          name
		}	
		required
        validations {
          name
          value
        }
      }
      hasThumbnailField
      access {
        admin
        storefront
      }
    }
    userErrors {
      field
      message
      code
    }
  }
}`

	type CreateMetaobjectDefinitionResponse struct {
		MetaobjectDefinitionCreate struct {
			CreatedDefinition *MetaobjectDefinition `json:"metaobjectDefinition"`
			UserErrors        UserErrors            `json:"userErrors"`
		} `json:"metaobjectDefinitionCreate"`
	}
	var gqlResp CreateMetaobjectDefinitionResponse
	err := c.shopifyClient.GraphQL.Query(ctx, query, variables, &gqlResp)
	if err != nil {
		return nil, err
	}
	if err := gqlResp.MetaobjectDefinitionCreate.UserErrors.Error(); err != nil {
		return nil, err
	}
	return gqlResp.MetaobjectDefinitionCreate.CreatedDefinition, nil
}

type GetMetaobjectDefinitionResponse struct {
	MetaobjectDefinition *MetaobjectDefinition `json:"metaobjectDefinition"`
}

func (c *Client) GetMetaobjectDefinition(ctx context.Context, id string) (*MetaobjectDefinition, error) {
	variables := map[string]interface{}{"id": id}
	query := `
query metaobjectDefinition($id: ID!) {
  metaobjectDefinition(id: $id) {
    id
    type
    name
    description
    displayNameKey
    fieldDefinitions {	
      key
      name
      description
      type {
        category
        name
      }	
      required
      validations {
        name
        value
      }
    }
    hasThumbnailField
    access {
      admin
      storefront
    }
  }
}
`

	var gqlResp GetMetaobjectDefinitionResponse
	err := retryGraphQLRead(ctx, c.graphQLReadRetryDelays, func() error {
		gqlResp = GetMetaobjectDefinitionResponse{}
		return c.shopifyClient.GraphQL.Query(ctx, query, variables, &gqlResp)
	})
	if err != nil {
		return nil, err
	}
	return gqlResp.MetaobjectDefinition, nil
}

type MetaobjectDefinitionUpdateInput struct {
	Name             string                                     `json:"name"`
	Description      *string                                    `json:"description,omitempty"`
	DisplayNameKey   *string                                    `json:"displayNameKey,omitempty"`
	FieldDefinitions []*MetaobjectFieldDefinitionOperationInput `json:"fieldDefinitions"`
	Access           *MetaobjectAccess                          `json:"access,omitempty"`
}

type MetaobjectFieldDefinitionOperationInput struct {
	Create *MetaobjectFieldDefinitionCreateInput `json:"create,omitempty"`
	Update *MetaobjectFieldDefinitionUpdateInput `json:"update,omitempty"`
	Delete *MetaobjectFieldDefinitionDeleteInput `json:"delete,omitempty"`
}

type MetaobjectFieldDefinitionUpdateInput struct {
	Key         string                           `json:"key"`
	Name        *string                          `json:"name"`
	Description *string                          `json:"description"`
	Required    bool                             `json:"required"`
	Validations []*MetafieldDefinitionValidation `json:"validations"`
}

type MetaobjectFieldDefinitionDeleteInput struct {
	Key string `json:"key"`
}

func (c *Client) UpdateMetaobjectDefinition(ctx context.Context, id string, input *MetaobjectDefinitionUpdateInput) (*MetaobjectDefinition, error) {
	variables := map[string]interface{}{"id": id, "definition": input}
	query := `
mutation UpdateMetaobjectDefinition($id: ID!, $definition: MetaobjectDefinitionUpdateInput!) {
  metaobjectDefinitionUpdate(id: $id, definition: $definition) {
    metaobjectDefinition {
      id
      type
      name
      description
      displayNameKey
      fieldDefinitions {	
	  	key
	  	name
	  	description
	  	type {
	  	  category
          name
	  	}	
	  	required
        validations {
          name
          value
        }
      }
      hasThumbnailField
      access {
        admin
        storefront
      }
    }
    userErrors {
      field
      message
      code
    }
  }
}`

	type UpdateMetaobjectDefinitionResponse struct {
		MetaobjectDefinitionUpdate struct {
			UpdatedDefinition *MetaobjectDefinition `json:"metaobjectDefinition"`
			UserErrors        UserErrors            `json:"userErrors"`
		} `json:"metaobjectDefinitionUpdate"`
	}

	var gqlResp UpdateMetaobjectDefinitionResponse
	err := c.shopifyClient.GraphQL.Query(ctx, query, variables, &gqlResp)
	if err != nil {
		return nil, err
	}
	if err := gqlResp.MetaobjectDefinitionUpdate.UserErrors.Error(); err != nil {
		return nil, err
	}
	return gqlResp.MetaobjectDefinitionUpdate.UpdatedDefinition, nil
}

func (c *Client) DeleteMetaobjectDefinition(ctx context.Context, id string) error {
	variables := map[string]interface{}{"id": id}
	query := `
mutation DeleteMetaobjectDefinition($id: ID!) {
  metaobjectDefinitionDelete(id: $id) {
	deletedId
    userErrors {
      field
      message
      code
    }
  }
}`

	type DeleteMetaobjectDefinitionResponse struct {
		MetaobjectDefinitionDelete struct {
			DeletedDefinitionID string `json:"deletedId"`
			UserErrors          UserErrors
		} `json:"metaobjectDefinitionDelete"`
	}
	var gqlResp DeleteMetaobjectDefinitionResponse
	err := c.shopifyClient.GraphQL.Query(ctx, query, variables, &gqlResp)
	if err != nil {
		return err
	}
	if err := gqlResp.MetaobjectDefinitionDelete.UserErrors.Error(); err != nil {
		return err
	}
	return nil
}
