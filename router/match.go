package router

import (
	"regexp"
)

func matchMethod(method, target string) bool {
	if "*" == target {
		return true
	}

	if target == method {
		return true
	}

	return false
}

func matchMethods(method string, target []string) bool {
	for _, v := range target {
		if matchMethod(method, v) {
			return true
		}
	}
	return false
}

func matchPath(path, target string) bool {

	res, err := regexp.MatchString(target, path)

	if err != nil {
		return false
	}

	return res
}
