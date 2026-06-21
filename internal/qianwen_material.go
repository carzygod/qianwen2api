package internal

import (
	"encoding/json"
	"fmt"
	"strings"
)

type qwenImageResource struct {
	ID         string `json:"id,omitempty"`
	URL        string `json:"url,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	FileFormat string `json:"file_format,omitempty"`
	FileName   string `json:"file_name,omitempty"`
	FileSize   int64  `json:"file_size,omitempty"`
}

func extractQwenImageResources(value interface{}) []qwenImageResource {
	var generic interface{}
	raw, err := json.Marshal(value)
	if err != nil || json.Unmarshal(raw, &generic) != nil {
		return nil
	}
	var out []qwenImageResource
	collectQwenImageResources(generic, &out)
	return dedupeQwenImageResources(out)
}

func resolveQwenVideoInputResources(req VideoGenerationRequest) ([]qwenImageResource, error) {
	var resources []qwenImageResource
	if req.Metadata != nil {
		resources = append(resources, extractQwenImageResources(req.Metadata)...)
	}

	sources := []string{
		req.FirstFrameImage,
		req.ImageURL,
		req.Image,
		req.FileID,
	}
	sources = append(sources, req.ReferenceImages...)

	for _, source := range dedupeStrings(sources) {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		resource, ok, err := resolveQwenImageResourceInput(source)
		if err != nil {
			return nil, err
		}
		if ok {
			resources = append(resources, resource)
		}
	}
	return dedupeQwenImageResources(resources), nil
}

func resolveQwenImageResourceInput(source string) (qwenImageResource, bool, error) {
	if resource, ok := parseQwenImageResourceString(source); ok {
		if strings.TrimSpace(resource.ID) != "" {
			return resource, true, nil
		}
	}
	if AppStore != nil {
		resource, err := AppStore.FindQianwenImageResource(source)
		if err != nil {
			return qwenImageResource{}, false, err
		}
		if resource != nil && strings.TrimSpace(resource.ID) != "" {
			return *resource, true, nil
		}
	}
	if isHTTPURL(source) || strings.HasPrefix(strings.TrimSpace(source), "data:") || looksLikeBase64Image(source) {
		return qwenImageResource{}, false, fmt.Errorf("qianwen.com video image input requires a Qianwen material id; generate the image through QIANWEN-WEB-01 first or pass metadata.qianwen_material_id / metadata.qwen_resource")
	}
	return qwenImageResource{ID: strings.TrimSpace(source)}, true, nil
}

func parseQwenImageResourceString(source string) (qwenImageResource, bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return qwenImageResource{}, false
	}
	for _, prefix := range []string{"qianwen_material_id:", "qianwen-resource:", "qianwen_resource:"} {
		if strings.HasPrefix(source, prefix) {
			id := strings.TrimSpace(strings.TrimPrefix(source, prefix))
			if id != "" {
				return qwenImageResource{ID: id}, true
			}
		}
	}
	if !strings.HasPrefix(source, "{") && !strings.HasPrefix(source, "[") {
		return qwenImageResource{}, false
	}
	var value interface{}
	if json.Unmarshal([]byte(source), &value) != nil {
		return qwenImageResource{}, false
	}
	resources := extractQwenImageResources(value)
	if len(resources) == 0 {
		return qwenImageResource{}, false
	}
	return resources[0], true
}

func collectQwenImageResources(value interface{}, out *[]qwenImageResource) {
	switch typed := value.(type) {
	case map[string]interface{}:
		if resource, ok := qwenImageResourceFromMap(typed); ok {
			*out = append(*out, resource)
		}
		for _, nested := range typed {
			collectQwenImageResources(nested, out)
		}
	case []interface{}:
		for _, nested := range typed {
			collectQwenImageResources(nested, out)
		}
	}
}

func qwenImageResourceFromMap(m map[string]interface{}) (qwenImageResource, bool) {
	if nested, ok := firstNestedValue(m, "qwen_resource", "qianwen_resource", "resource_info", "resource"); ok {
		if nestedMap, ok := nested.(map[string]interface{}); ok {
			if resource, ok := qwenImageResourceFromMap(nestedMap); ok {
				return resource, true
			}
		}
	}
	if infos, ok := firstNestedValue(m, "resource_infos", "resourceInfos"); ok {
		if list, ok := infos.([]interface{}); ok && len(list) > 0 {
			if nestedMap, ok := list[0].(map[string]interface{}); ok {
				if resource, ok := qwenImageResourceFromMap(nestedMap); ok {
					return resource, true
				}
			}
		}
	}
	if attachments, ok := firstNestedValue(m, "attachments"); ok {
		if list, ok := attachments.([]interface{}); ok && len(list) > 0 {
			if nestedMap, ok := list[0].(map[string]interface{}); ok {
				if resource, ok := qwenImageResourceFromMap(nestedMap); ok {
					return resource, true
				}
			}
		}
	}

	explicitID := firstStringValue(m, "qianwen_material_id", "materialId", "material_id", "resourceId", "resource_id", "imageResourceId", "ws_gid")
	url := firstStringValue(m, "imageResourceUrl", "material_cdn_url", "material_url", "image_url", "public_url", "upstream_url", "url")
	resourceContext := explicitID != "" || url != "" || hasAnyKey(m, "fileFormat", "file_format", "fileName", "file_name") || strings.EqualFold(firstStringValue(m, "type"), "image")
	if explicitID == "" && resourceContext {
		explicitID = firstStringValue(m, "id")
	}
	resource := qwenImageResource{
		ID:         explicitID,
		URL:        url,
		Width:      firstIntValue(m, "imageResourceWidth", "width"),
		Height:     firstIntValue(m, "imageResourceHeight", "height"),
		FileFormat: firstStringValue(m, "fileFormat", "file_format"),
		FileName:   firstStringValue(m, "fileName", "file_name", "name"),
		FileSize:   firstInt64Value(m, "fileSize", "file_size", "size"),
	}
	if strings.TrimSpace(resource.ID) == "" && strings.TrimSpace(resource.URL) == "" {
		return qwenImageResource{}, false
	}
	return resource, true
}

func qwenVideoAttachments(resources []qwenImageResource) []map[string]string {
	attachments := make([]map[string]string, 0, len(resources))
	for _, resource := range resources {
		id := strings.TrimSpace(resource.ID)
		if id == "" {
			continue
		}
		attachments = append(attachments, map[string]string{
			"type":       "image",
			"materialId": id,
		})
	}
	return attachments
}

func addQwenImageMessagesToPayload(payload map[string]interface{}, resources []qwenImageResource) {
	if len(resources) == 0 {
		return
	}
	messages, ok := payload["messages"].([]map[string]interface{})
	if !ok {
		return
	}
	imageMessages := make([]map[string]interface{}, 0, len(resources))
	for _, resource := range resources {
		if strings.TrimSpace(resource.ID) == "" {
			continue
		}
		content := strings.TrimSpace(resource.URL)
		if content == "" {
			content = resource.ID
		}
		imageMessages = append(imageMessages, map[string]interface{}{
			"mime_type": "image/url",
			"content":   content,
			"meta_data": map[string]interface{}{
				"resource_infos": []map[string]interface{}{qwenImageResourceMap(resource)},
			},
			"status": "complete",
		})
	}
	if len(imageMessages) == 0 {
		return
	}
	payload["messages"] = append(imageMessages, messages...)
}

func qwenImageResourceMap(resource qwenImageResource) map[string]interface{} {
	out := map[string]interface{}{}
	if resource.ID != "" {
		out["id"] = resource.ID
	}
	if resource.URL != "" {
		out["url"] = resource.URL
	}
	if resource.Width > 0 {
		out["width"] = resource.Width
	}
	if resource.Height > 0 {
		out["height"] = resource.Height
	}
	if resource.FileFormat != "" {
		out["file_format"] = resource.FileFormat
	}
	if resource.FileName != "" {
		out["file_name"] = resource.FileName
	}
	if resource.FileSize > 0 {
		out["file_size"] = resource.FileSize
	}
	return out
}

func dedupeQwenImageResources(resources []qwenImageResource) []qwenImageResource {
	out := make([]qwenImageResource, 0, len(resources))
	seen := map[string]bool{}
	for _, resource := range resources {
		resource.ID = strings.TrimSpace(resource.ID)
		resource.URL = strings.TrimSpace(resource.URL)
		if resource.ID == "" && resource.URL == "" {
			continue
		}
		key := resource.ID
		if key == "" {
			key = resource.URL
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, resource)
	}
	return out
}

func dedupeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func resourceForURL(resources []qwenImageResource, rawURL string) (qwenImageResource, bool) {
	rawURL = strings.TrimSpace(rawURL)
	for _, resource := range resources {
		if strings.TrimSpace(resource.URL) == rawURL {
			return resource, true
		}
	}
	return qwenImageResource{}, false
}

func isHTTPURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func firstNestedValue(m map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok && value != nil {
			return value, true
		}
	}
	return nil, false
}

func hasAnyKey(m map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if _, ok := m[key]; ok {
			return true
		}
	}
	return false
}

func firstStringValue(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if text := qwenValueString(value); text != "" {
				return text
			}
		}
	}
	return ""
}

func firstIntValue(m map[string]interface{}, keys ...string) int {
	return int(firstInt64Value(m, keys...))
}

func firstInt64Value(m map[string]interface{}, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			switch typed := value.(type) {
			case int:
				return int64(typed)
			case int64:
				return typed
			case float64:
				return int64(typed)
			case json.Number:
				n, _ := typed.Int64()
				return n
			case string:
				var n json.Number = json.Number(strings.TrimSpace(typed))
				if parsed, err := n.Int64(); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}

func qwenValueString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return ""
	}
}
