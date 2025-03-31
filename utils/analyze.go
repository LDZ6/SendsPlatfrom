package utils

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"strings"
)

func JSONToMap(str string) (map[string]any, error) {

	var tempMap = make(map[string]any)

	err := json.Unmarshal([]byte(str), &tempMap)

	if err != nil {
		return tempMap, err
	}

	return tempMap, nil
}

func RemoveInvalidUTF8(result *[]string, s string) {
	var start int
	fmt.Println(s)
	for i := 0; i < len(s); i++ {
		if (s[i:i+1] < "0" || s[i:i+1] > "9") && s[i:i+1] != "/" && s[i:i+1] != "." {
			if start != i {
				*result = append(*result, s[start:i])
				start = i
				start++
			} else {
				start++
			}
		}
	}
	*result = append(*result, s[start:len(s)])
}

func Traverse(n *html.Node, result *[]string) {
	// 检查当前节点是否为<td>元素
	if n.Type == html.ElementNode && n.Data == "td" {
		// 检查<td>元素的class属性值是否为所需字段所在元素的class（例如：b1）
		for _, attr := range n.Attr {
			if attr.Key == "class" && attr.Val == "fh tac bw f10-0 pl2 b1" {
				// 输出字段内容
				*result = append(*result, n.FirstChild.Data)
			} else if attr.Key == "class" && attr.Val == "fh bw f10-0 b1" {
				*result = append(*result, "总计")
				RemoveInvalidUTF8(result, n.FirstChild.Data)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		Traverse(c, result)
	}
}

func GetCurrentSessionID(scriptContent string) string {
	startIndex := strings.Index(scriptContent, "currentSessionID")
	if startIndex == -1 {
		return ""
	}
	startIndex += len("currentSessionID= '") + 1
	endIndex := strings.IndexAny(scriptContent[startIndex:], "';\r\n")
	if endIndex == -1 {
		endIndex = len(scriptContent)
	} else {
		endIndex += startIndex
	}
	return scriptContent[startIndex:endIndex]
}
