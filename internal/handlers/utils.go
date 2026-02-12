package handlers

import (
	"regexp"
	"strings"
)

func generateSlug(name, city string) string {
	// Combine name and city
	base := name + "-" + city
	
	// Lowercase
	slug := strings.ToLower(base)
	
	// Replace non-alphanumeric with hyphens
	reg := regexp.MustCompile("[^a-z0-9]+")
	slug = reg.ReplaceAllString(slug, "-")
	
	// Trim hyphens
	slug = strings.Trim(slug, "-")
	
	return slug
}
