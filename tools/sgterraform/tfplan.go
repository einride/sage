package sgterraform

// This file contains needed functionality and definitions from github.com/hashicorp/terraform-json to parse the
// tf json plan.

// Plan contains a simplified definition of the json generated when using the -json flag for terraform plan.
// It only includes the fields needed to generate the plan summery used in CommentOnPRWithPlanSummarized.
// The full plan definition can be found in github.com/hashicorp/terraform-json.
type Plan struct {
	ResourceChanges []*struct {
		// The resource type, example: "aws_instance" for aws_instance.foo.
		Type string `json:"type,omitempty"`

		// The resource name, example: "foo" for aws_instance.foo.
		Name string `json:"name,omitempty"`

		// The data describing the change that will be made to this object.
		Change *struct {
			// The action to be carried out by this change.
			Actions Actions `json:"actions,omitempty"`
		} `json:"change,omitempty"`
	} `json:"resource_changes,omitempty"`
}

// Action is a valid action type for a resource change.
//
// Note that a singular Action is not telling of a full resource
// change operation. Certain resource actions, such as replacement,
// are a composite of more than one type. See the Actions type and
// its helpers for more information.
type Action string

const (
	// ActionNoop denotes a no-op operation.
	ActionNoop Action = "no-op"

	// ActionCreate denotes a create operation.
	ActionCreate Action = "create"

	// ActionRead denotes a read operation.
	ActionRead Action = "read"

	// ActionUpdate denotes an update operation.
	ActionUpdate Action = "update"

	// ActionDelete denotes a delete operation.
	ActionDelete Action = "delete"

	// ActionForget denotes a forget operation.
	ActionForget Action = "forget"
)

// Actions denotes a valid change type.
type Actions []Action

// NoOp is true if this set of Actions denotes a no-op.
func (a Actions) NoOp() bool {
	if len(a) != 1 {
		return false
	}

	return a[0] == ActionNoop
}

// Create is true if this set of Actions denotes creation of a new
// resource.
func (a Actions) Create() bool {
	if len(a) != 1 {
		return false
	}

	return a[0] == ActionCreate
}

// Read is true if this set of Actions denotes a read operation only.
func (a Actions) Read() bool {
	if len(a) != 1 {
		return false
	}

	return a[0] == ActionRead
}

// Update is true if this set of Actions denotes an update operation.
func (a Actions) Update() bool {
	if len(a) != 1 {
		return false
	}

	return a[0] == ActionUpdate
}

// Delete is true if this set of Actions denotes resource removal.
func (a Actions) Delete() bool {
	if len(a) != 1 {
		return false
	}

	return a[0] == ActionDelete
}

// DestroyBeforeCreate is true if this set of Actions denotes a
// destroy-before-create operation. This is the standard resource
// replacement method.
func (a Actions) DestroyBeforeCreate() bool {
	if len(a) != 2 {
		return false
	}

	return a[0] == ActionDelete && a[1] == ActionCreate
}

// CreateBeforeDestroy is true if this set of Actions denotes a
// create-before-destroy operation, usually the result of replacement
// to a resource that has the create_before_destroy lifecycle option
// set.
func (a Actions) CreateBeforeDestroy() bool {
	if len(a) != 2 {
		return false
	}

	return a[0] == ActionCreate && a[1] == ActionDelete
}

// Replace is true if this set of Actions denotes a valid replacement
// operation.
func (a Actions) Replace() bool {
	return a.DestroyBeforeCreate() || a.CreateBeforeDestroy()
}

// Forget is true if this set of Actions denotes a forget operation.
func (a Actions) Forget() bool {
	if len(a) != 1 {
		return false
	}

	return a[0] == ActionForget
}
