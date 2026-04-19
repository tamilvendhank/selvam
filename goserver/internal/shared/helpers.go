package shared

import "go.mongodb.org/mongo-driver/bson/primitive"

func StringPtr(value string) *string {
	return &value
}

func IntPtr(value int) *int {
	return &value
}

func DerefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func NormalizeJSONValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case primitive.M:
		output := make(map[string]any, len(typed))
		for key, item := range typed {
			output[key] = NormalizeJSONValue(item)
		}
		return output
	case primitive.D:
		output := make(map[string]any, len(typed))
		for _, item := range typed {
			output[item.Key] = NormalizeJSONValue(item.Value)
		}
		return output
	case primitive.A:
		output := make([]any, len(typed))
		for index, item := range typed {
			output[index] = NormalizeJSONValue(item)
		}
		return output
	case []any:
		output := make([]any, len(typed))
		for index, item := range typed {
			output[index] = NormalizeJSONValue(item)
		}
		return output
	case map[string]any:
		output := make(map[string]any, len(typed))
		for key, item := range typed {
			output[key] = NormalizeJSONValue(item)
		}
		return output
	default:
		return value
	}
}

func MapFromAny(value any) map[string]any {
	normalized := NormalizeJSONValue(value)
	if normalized == nil {
		return nil
	}

	mapped, _ := normalized.(map[string]any)
	return mapped
}
