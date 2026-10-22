package engine

import (
	"bufio"
	"os"
	"strings"
)

// ParseSimpleScript mem-parse format DSL/script bergaya Zeno/PHP slots menjadi AST Tree
// Format yang didukung:
// slot_name: value {
//     child_slot: value
// }
func ParseScript(input string) (*Node, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	root := &Node{Name: "root"}
	stack := []*Node{root}
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		if line == "}" {
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
			continue
		}

		hasOpenBrace := strings.HasSuffix(line, "{")
		cleanLine := strings.TrimSuffix(line, "{")
		cleanLine = strings.TrimSpace(cleanLine)

		parts := strings.SplitN(cleanLine, ":", 2)
		slotName := strings.TrimSpace(parts[0])
		var slotVal interface{} = nil

		if len(parts) > 1 {
			valStr := strings.TrimSpace(parts[1])
			valStr = strings.Trim(valStr, "\"'")
			slotVal = valStr
		}

		node := &Node{
			Name:  slotName,
			Value: slotVal,
			Line:  lineNum,
		}

		currParent := stack[len(stack)-1]
		node.Parent = currParent
		currParent.Children = append(currParent.Children, node)

		if hasOpenBrace {
			stack = append(stack, node)
		}
	}

	return root, scanner.Err()
}

func ParseFile(path string) (*Node, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseScript(string(bytes))
}
