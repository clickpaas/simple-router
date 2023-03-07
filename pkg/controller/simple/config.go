package simple

import "strings"

type tagItem struct{
	firstLevel string
	secondLevel string
}

type Config struct {
	MultiTag []string
	NumTags int
	WhiteDplList []string
}

// parseTags parse config ,get tagItems from config
func(c *Config)parseTags()[]tagItem{
	var tagItems []tagItem
	for _, key := range c.MultiTag{
		keyList := strings.Split(key, "|")
		tagItems = append(tagItems, tagItem{firstLevel: keyList[0], secondLevel: keyList[1]})
	}
	return tagItems
}
