package nerdgraph

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

func ResolveSchemaTypes(schema Schema, typeNames []string) (map[string]string, error) {
	typeKeeper := make(map[string]string)

	for _, typeName := range typeNames {
		typeGenResult, err := TypeGen(schema, typeName)
		if err != nil {
			log.Errorf("error while generating type %s: %s", typeName, err)
		}

		for k, v := range typeGenResult {
			typeKeeper[k] = v
		}
	}

	return typeKeeper, nil
}

func handleEnumType(schema Schema, t SchemaType) map[string]string {
	types := make(map[string]string)

	// output collects each line of a struct type
	output := []string{}
	output = append(output, fmt.Sprintf("type %s int ", t.Name))
	output = append(output, "")

	output = append(output, "const (")
	for i, v := range t.EnumValues {
		if i == 0 {
			output = append(output, fmt.Sprintf("\t%s = iota", v.Name))
		} else {
			output = append(output, fmt.Sprintf("\t%s", v.Name))
		}

	}
	output = append(output, ")")
	output = append(output, "")

	types[t.Name] = strings.Join(output, "\n")

	return types
}

func kindTree(f SchemaInputValue) []string {
	tree := []string{}

	if f.Type.Kind != "" {
		tree = append(tree, f.Type.Kind)
	}

	if f.Type.OfType.Kind != "" {
		tree = append(tree, f.Type.OfType.Kind)
	}

	if f.Type.OfType.OfType.Kind != "" {
		tree = append(tree, f.Type.OfType.OfType.Kind)
	}

	if f.Type.OfType.OfType.OfType.Kind != "" {
		tree = append(tree, f.Type.OfType.OfType.OfType.Kind)
	}

	if f.Type.OfType.OfType.OfType.OfType.Kind != "" {
		tree = append(tree, f.Type.OfType.OfType.OfType.OfType.Kind)
	}

	if f.Type.OfType.OfType.OfType.OfType.OfType.Kind != "" {
		tree = append(tree, f.Type.OfType.OfType.OfType.OfType.OfType.Kind)
	}

	if f.Type.OfType.OfType.OfType.OfType.OfType.OfType.Kind != "" {
		tree = append(tree, f.Type.OfType.OfType.OfType.OfType.OfType.OfType.Kind)
	}

	return tree
}

func nameTree(f SchemaInputValue) []string {
	tree := []string{}

	if f.Type.Name != "" {
		tree = append(tree, f.Type.Name)
	}

	if f.Type.OfType.Name != "" {
		tree = append(tree, f.Type.OfType.Name)
	}

	if f.Type.OfType.OfType.Name != "" {
		tree = append(tree, f.Type.OfType.OfType.Name)
	}

	if f.Type.OfType.OfType.OfType.Name != "" {
		tree = append(tree, f.Type.OfType.OfType.OfType.Name)
	}

	if f.Type.OfType.OfType.OfType.OfType.Name != "" {
		tree = append(tree, f.Type.OfType.OfType.OfType.OfType.Name)
	}

	if f.Type.OfType.OfType.OfType.OfType.OfType.Name != "" {
		tree = append(tree, f.Type.OfType.OfType.OfType.OfType.OfType.Name)
	}

	if f.Type.OfType.OfType.OfType.OfType.OfType.OfType.Name != "" {
		tree = append(tree, f.Type.OfType.OfType.OfType.OfType.OfType.OfType.Name)
	}

	return tree
}

func removeNonNullValues(tree []string) []string {
	a := []string{}

	for _, x := range tree {
		if x != "NON_NULL" {
			a = append(a, x)
		}
	}

	return a
}

// fieldTypeFromTypeRef resolves the given SchemaInputValue into a field name to use on a go struct.
func fieldTypeFromTypeRef(f SchemaInputValue) (string, bool, error) {

	switch t := nameTree(f)[0]; t {
	case "String":
		return "string", false, nil
	case "Int":
		return "int", false, nil
	case "Boolean":
		return "bool", false, nil
	case "Float":
		return "float64", false, nil
	case "ID":
		// ID is a nested object, but behaves like an integer.  This may be true of other SCALAR types as well, so logic here could potentially be moved.
		return "int", false, nil
	case "":
		return "", true, fmt.Errorf("empty field name: %+v", f)
	default:
		return t, true, nil
	}
}

// handleObjectType will operate on a SchemaType who's Kind is OBJECT or INPUT_OBJECT.
func handleObjectType(schema Schema, t SchemaType) map[string]string {
	types := make(map[string]string)
	var err error
	recurse := false

	// output collects each line of a struct type
	output := []string{}

	output = append(output, fmt.Sprintf("type %s struct {", t.Name))

	// Fill in the struct fields for an input type
	for _, f := range t.InputFields {
		var fieldType string

		log.Debugf("handling kind %s: %+v\n\n", f.Type.Kind, f)
		fieldType, recurse, err = fieldTypeFromTypeRef(f)
		if err != nil {
			// If we have an error, then we don't know how to handle the type to
			// determine the field name.  This indicates that
			log.Errorf("error resolving first non-empty name from field: %s: %s", f, err)
		}

		if recurse {
			// The name of the nested sub-type.  We take the first value here as the root name for the nested type.
			subTName := nameTree(f)[0]

			subT, err := typeByName(schema, subTName)
			if err != nil {
				log.Warnf("non_null: unhandled type: %+v\n", f)
				continue
			}

			// Determnine if we need to resolve the sub type, or if it already
			// exists in the map.
			if _, ok := types[subT.Name]; !ok {
				result, err := TypeGen(schema, subT.Name)
				if err != nil {
					log.Errorf("ERROR while resolving sub type %s: %s\n", subT.Name, err)
				}

				log.Debugf("resolved type result:\n%+v\n", result)

				for k, v := range result {
					if _, ok := types[k]; !ok {
						types[k] = v
					}
				}
			}

			fieldType = subT.Name
		}

		fieldName := strings.Title(f.Name)

		// The prefix is used to ensure that we handle LIST or slices correctly.
		fieldTypePrefix := ""

		if removeNonNullValues(kindTree(f))[0] == "LIST" {
			fieldTypePrefix = "[]"
		}

		// Include some documentation
		if f.Description != "" {
			output = append(output, fmt.Sprintf("\t /* %s */", f.Description))
		}

		fieldTags := fmt.Sprintf("`json:\"%s\"`", f.Name)

		output = append(output, fmt.Sprintf("\t %s %s%s %s", fieldName, fieldTypePrefix, fieldType, fieldTags))
		output = append(output, "")
	}

	for _, f := range t.EnumValues {
		log.Debugf("\n\nEnums: %+v\n", f)
	}

	for _, f := range t.Fields {
		log.Debugf("\n\nFields: %+v\n", f)
	}

	// Close the struct
	output = append(output, "}\n")
	types[t.Name] = strings.Join(output, "\n")

	return types
}

// TypeGen is the mother type generator.
func TypeGen(schema Schema, typeName string) (map[string]string, error) {

	// The total known types.  Keyed by the typeName, and valued as the string
	// output that one would write to a file where Go structs are kept.
	types := make(map[string]string)

	t, err := typeByName(schema, typeName)
	if err != nil {
		log.Error(err)
	}

	log.Infof("starting on %s: %+v", typeName, t.Kind)

	// To store the results from the single
	results := make(map[string]string)

	if t.Kind == "INPUT_OBJECT" || t.Kind == "OBJECT" {
		results = handleObjectType(schema, *t)
	} else if t.Kind == "ENUM" {
		results = handleEnumType(schema, *t)
	} else {
		log.Warnf("WARN: unhandled object Kind: %s\n", t.Kind)
	}

	for k, v := range results {
		types[k] = v
	}

	// return strings.Join(output, "\n"), nil
	return types, nil
}

func typeByName(schema Schema, typeName string) (*SchemaType, error) {
	log.Debugf("looking for typeName: %s", typeName)

	for _, t := range schema.Types {
		if t.Name == typeName {
			return t, nil
		}
	}

	return nil, fmt.Errorf("type by name %s not found", typeName)
}