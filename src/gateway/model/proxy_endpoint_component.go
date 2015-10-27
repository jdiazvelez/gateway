package model

import (
	"fmt"
	aperrors "gateway/errors"
	apsql "gateway/sql"

	"github.com/jmoiron/sqlx/types"
)

const (
	ProxyEndpointComponentTypeSingle = "single"
	ProxyEndpointComponentTypeMulti  = "multi"
	ProxyEndpointComponentTypeJS     = "js"
)

type ProxyEndpointComponent struct {
	ID                    int64                          `json:"id,omitempty"`
	Conditional           string                         `json:"conditional"`
	ConditionalPositive   bool                           `json:"conditional_positive" db:"conditional_positive"`
	Type                  string                         `json:"type"`
	BeforeTransformations []*ProxyEndpointTransformation `json:"before,omitempty"`
	AfterTransformations  []*ProxyEndpointTransformation `json:"after,omitempty"`
	Call                  *ProxyEndpointCall             `json:"call,omitempty"`
	Calls                 []*ProxyEndpointCall           `json:"calls,omitempty"`
	Data                  types.JsonText                 `json:"data,omitempty"`
	SharedComponentID     int64                          `json:"shared_component_id,omitempty"`
}

// Validate validates the model.
func (c *ProxyEndpointComponent) Validate() aperrors.Errors {
	if c.SharedComponentID != 0 {
		return c.validateShared()
	}

	return c.validateNonShared()
}

func (c *ProxyEndpointComponent) validateShared() aperrors.Errors {
	// This is totally redundant for now.
	return c.validateNonShared()
}

func (c *ProxyEndpointComponent) validateNonShared() aperrors.Errors {
	errors := make(aperrors.Errors)

	switch c.Type {
	case ProxyEndpointComponentTypeSingle:
	case ProxyEndpointComponentTypeMulti:
	case ProxyEndpointComponentTypeJS:
	default:
		errors.Add("type", "must be one of 'single', or 'multi', or 'js'")
	}

	errors.AddErrors(c.validateTransformations())

	return errors
}

func (c *ProxyEndpointComponent) validateTransformations() aperrors.Errors {
	errors := make(aperrors.Errors)

	for i, t := range c.BeforeTransformations {
		tErrors := t.Validate()
		if !tErrors.Empty() {
			errors.Add("before", fmt.Sprintf("%d is invalid: %v", i, tErrors))
		}
	}

	for i, t := range c.AfterTransformations {
		tErrors := t.Validate()
		if !tErrors.Empty() {
			errors.Add("after", fmt.Sprintf("%d is invalid: %v", i, tErrors))
		}
	}

	return errors
}

// AllCalls provides a common interface to iterate through single and multi-call
// components' calls.
func (c *ProxyEndpointComponent) AllCalls() []*ProxyEndpointCall {
	if c.Type == ProxyEndpointComponentTypeSingle {
		return []*ProxyEndpointCall{c.Call}
	}

	return c.Calls
}

// AllProxyEndpointsForAPIIDAndAccountID returns all components of an endpoint.
func AllProxyEndpointComponentsForEndpointID(db *apsql.DB, endpointID int64) ([]*ProxyEndpointComponent, error) {
	components := []*ProxyEndpointComponent{}
	err := db.Select(&components,
		`SELECT
			id, conditional, conditional_positive, type, data, shared_component_id
		FROM proxy_endpoint_components
		WHERE endpoint_id = ?
		ORDER BY position ASC;`,
		endpointID)
	if err != nil {
		return nil, err
	}

	var componentIDs []int64
	componentsByID := make(map[int64]*ProxyEndpointComponent)
	for _, component := range components {
		componentIDs = append(componentIDs, component.ID)
		componentsByID[component.ID] = component
	}

	calls, err := AllProxyEndpointCallsForComponentIDs(db, componentIDs)
	if err != nil {
		return nil, err
	}

	var callIDs []int64
	callsByID := make(map[int64]*ProxyEndpointCall)
	for _, call := range calls {
		callIDs = append(callIDs, call.ID)
		callsByID[call.ID] = call
		component := componentsByID[call.ComponentID]
		switch component.Type {
		case ProxyEndpointComponentTypeSingle:
			component.Call = call
		case ProxyEndpointComponentTypeMulti:
			component.Calls = append(component.Calls, call)
		}
	}

	transforms, err := AllProxyEndpointTransformationsForComponentIDsAndCallIDs(db,
		componentIDs, callIDs)
	if err != nil {
		return nil, err
	}

	for _, transform := range transforms {
		if transform.ComponentID != nil {
			component := componentsByID[*transform.ComponentID]
			if transform.Before {
				component.BeforeTransformations = append(component.BeforeTransformations, transform)
			} else {
				component.AfterTransformations = append(component.AfterTransformations, transform)
			}
		} else if transform.CallID != nil {
			call := callsByID[*transform.CallID]
			if transform.Before {
				call.BeforeTransformations = append(call.BeforeTransformations, transform)
			} else {
				call.AfterTransformations = append(call.AfterTransformations, transform)
			}
		}
	}

	return components, err
}

// DeleteProxyEndpointComponentsWithEndpointIDAndNotInList
func DeleteProxyEndpointComponentsWithEndpointIDAndNotInList(tx *apsql.Tx,
	endpointID int64, validIDs []int64) error {

	args := []interface{}{endpointID}
	var validIDQuery string
	if len(validIDs) > 0 {
		validIDQuery = " AND id NOT IN (" + apsql.NQs(len(validIDs)) + ")"
		for _, id := range validIDs {
			args = append(args, id)
		}
	}
	_, err := tx.Exec(
		`DELETE FROM proxy_endpoint_components
		WHERE endpoint_id = ?`+validIDQuery+`;`,
		args...)
	return err
}

// Insert inserts the component into the database as a new row.
func (c *ProxyEndpointComponent) Insert(
	tx *apsql.Tx,
	endpointID, apiID int64,
	position int,
) error {
	data, err := marshaledForStorage(c.Data)
	if err != nil {
		return aperrors.NewWrapped("Marshaling component JSON", err)
	}

	c.ID, err = tx.InsertOne(
		`INSERT INTO proxy_endpoint_components
			(endpoint_id, conditional, conditional_positive,
			 position, type, data, shared_component_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		endpointID, c.Conditional, c.ConditionalPositive,
		position, c.Type, data, c.SharedComponentID)
	if err != nil {
		return aperrors.NewWrapped("Inserting component", err)
	}

	for position, transform := range c.BeforeTransformations {
		err = transform.InsertForComponent(tx, c.ID, true, position)
		if err != nil {
			return aperrors.NewWrapped("Inserting before transformation", err)
		}
	}
	for position, transform := range c.AfterTransformations {
		err = transform.InsertForComponent(tx, c.ID, false, position)
		if err != nil {
			return aperrors.NewWrapped("Inserting after transformation", err)
		}
	}

	switch c.Type {
	case ProxyEndpointComponentTypeSingle:
		if err = c.Call.Insert(tx, c.ID, apiID, 0); err != nil {
			return aperrors.NewWrapped("Inserting single call", err)
		}
	case ProxyEndpointComponentTypeMulti:
		for position, call := range c.Calls {
			if err = call.Insert(tx, c.ID, apiID, position); err != nil {
				return aperrors.NewWrapped("Inserting multi call", err)
			}
		}
	default:
	}

	return nil
}

// Update updates the component in place.
func (c *ProxyEndpointComponent) Update(tx *apsql.Tx, endpointID, apiID int64,
	position int) error {

	data, err := marshaledForStorage(c.Data)
	if err != nil {
		return err
	}

	err = tx.UpdateOne(
		`UPDATE proxy_endpoint_components
		SET
			conditional = ?,
			conditional_positive = ?,
			position = ?,
			type = ?,
			data = ?,
			shared_component_id = ?
		WHERE id = ? AND endpoint_id = ?;`,
		c.Conditional,
		c.ConditionalPositive,
		position,
		c.Type,
		data,
		c.SharedComponentID,
		c.ID,
		endpointID,
	)
	if err != nil {
		return err
	}

	var validTransformationIDs []int64
	for position, transformation := range c.BeforeTransformations {
		if transformation.ID == 0 {
			err = transformation.InsertForComponent(tx, c.ID, true, position)
			if err != nil {
				return err
			}
		} else {
			err = transformation.UpdateForComponent(tx, c.ID, true, position)
			if err != nil {
				return err
			}
		}
		validTransformationIDs = append(validTransformationIDs, transformation.ID)
	}
	for position, transformation := range c.AfterTransformations {
		if transformation.ID == 0 {
			err = transformation.InsertForComponent(tx, c.ID, false, position)
			if err != nil {
				return err
			}
		} else {
			err = transformation.UpdateForComponent(tx, c.ID, false, position)
			if err != nil {
				return err
			}
		}
		validTransformationIDs = append(validTransformationIDs, transformation.ID)
	}
	err = DeleteProxyEndpointTransformationsWithComponentIDAndNotInList(tx,
		c.ID, validTransformationIDs)
	if err != nil {
		return err
	}

	var validCallIDs []int64
	switch c.Type {
	case ProxyEndpointComponentTypeSingle:
		if c.Call.ID == 0 {
			err = c.Call.Insert(tx, c.ID, apiID, 0)
			if err != nil {
				return err
			}
		} else {
			err = c.Call.Update(tx, c.ID, apiID, 0)
			if err != nil {
				return err
			}
		}
		validCallIDs = append(validCallIDs, c.Call.ID)
	case ProxyEndpointComponentTypeMulti:
		for position, call := range c.Calls {
			if call.ID == 0 {
				err = call.Insert(tx, c.ID, apiID, position)
				if err != nil {
					return err
				}
			} else {
				err = call.Update(tx, c.ID, apiID, position)
				if err != nil {
					return err
				}
			}
			validCallIDs = append(validCallIDs, call.ID)
		}
	default:
	}

	err = DeleteProxyEndpointCallsWithComponentIDAndNotInList(tx,
		c.ID, validCallIDs)
	if err != nil {
		return err
	}

	return nil
}
