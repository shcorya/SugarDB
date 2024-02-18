package list

import (
	"context"
	"errors"
	"fmt"
	"github.com/echovault/echovault/src/utils"
	"math"
	"net"
	"slices"
	"strings"
)

func handleLLen(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := llenKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	key := keys[0]

	if !server.KeyExists(key) {
		// If key does not exist, return 0
		return []byte(":0\r\n\r\n"), nil
	}

	if _, err = server.KeyRLock(ctx, key); err != nil {
		return nil, err
	}
	defer server.KeyRUnlock(key)

	if list, ok := server.GetValue(key).([]interface{}); ok {
		return []byte(fmt.Sprintf(":%d\r\n\r\n", len(list))), nil
	}

	return nil, errors.New("LLEN command on non-list item")
}

func handleLIndex(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := lindexKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	key := keys[0]
	index, ok := utils.AdaptType(cmd[2]).(int)

	if !ok {
		return nil, errors.New("index must be an integer")
	}

	if !server.KeyExists(key) {
		return nil, errors.New("LINDEX command on non-list item")
	}

	if _, err = server.KeyRLock(ctx, key); err != nil {
		return nil, err
	}
	list, ok := server.GetValue(key).([]interface{})
	server.KeyRUnlock(key)

	if !ok {
		return nil, errors.New("LINDEX command on non-list item")
	}

	if !(index >= 0 && index < len(list)) {
		return nil, errors.New("index must be within list range")
	}

	return []byte(fmt.Sprintf("+%s\r\n\r\n", list[index])), nil
}

func handleLRange(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := lrangeKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	key := keys[0]
	start, startOk := utils.AdaptType(cmd[2]).(int)
	end, endOk := utils.AdaptType(cmd[3]).(int)

	if !startOk || !endOk {
		return nil, errors.New("start and end indices must be integers")
	}

	if !server.KeyExists(key) {
		return nil, errors.New("LRANGE command on non-list item")
	}

	if _, err = server.KeyRLock(ctx, key); err != nil {
		return nil, err
	}
	defer server.KeyRUnlock(key)

	list, ok := server.GetValue(key).([]interface{})
	if !ok {
		return nil, errors.New("LRANGE command on non-list item")
	}

	// Make sure start is within range
	if !(start >= 0 && start < len(list)) {
		return nil, errors.New("start index must be within list boundary")
	}

	// Make sure end is within range, or is -1 otherwise
	if !((end >= 0 && end < len(list)) || end == -1) {
		return nil, errors.New("end index must be within list range or -1")
	}

	var bytes []byte

	// If end is -1, read list from start to the end of the list
	if end == -1 {
		bytes = []byte("*" + fmt.Sprint(len(list)-int(start)) + "\r\n")
		for i := int(start); i < len(list); i++ {
			str := fmt.Sprintf("%v", list[i])
			bytes = append(bytes, []byte("$"+fmt.Sprint(len(str))+"\r\n"+str+"\r\n")...)
		}
		bytes = append(bytes, []byte("\r\n")...)
		return bytes, nil
	}

	// Make sure start and end are not equal to each other
	if start == end {
		return nil, errors.New("start and end indices cannot be equal")
	}

	// If end is not -1:
	//	1) If end is larger than start, return slice from start -> end
	//	2) If end is smaller than start, return slice from end -> start
	bytes = []byte("*" + fmt.Sprint(int(math.Abs(float64(start-end)))+1) + "\r\n")

	i := start
	j := end + 1
	if start > end {
		j = end - 1
	}

	for i != j {
		str := fmt.Sprintf("%v", list[i])
		bytes = append(bytes, []byte("$"+fmt.Sprint(len(str))+"\r\n"+str+"\r\n")...)
		if start < end {
			i++
		} else {
			i--
		}

	}

	bytes = append(bytes, []byte("\r\n")...)

	return bytes, nil
}

func handleLSet(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := lsetKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	key := keys[0]

	index, ok := utils.AdaptType(cmd[2]).(int)
	if !ok {
		return nil, errors.New("index must be an integer")
	}

	if !server.KeyExists(key) {
		return nil, errors.New("LSET command on non-list item")
	}

	if _, err = server.KeyLock(ctx, key); err != nil {
		return nil, err
	}
	defer server.KeyUnlock(key)

	list, ok := server.GetValue(key).([]interface{})
	if !ok {
		return nil, errors.New("LSET command on non-list item")
	}

	if !(index >= 0 && index < len(list)) {
		return nil, errors.New("index must be within list range")
	}

	list[index] = utils.AdaptType(cmd[3])
	server.SetValue(ctx, key, list)

	return []byte(utils.OK_RESPONSE), nil
}

func handleLTrim(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := ltrimKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	key := keys[0]
	start, startOk := utils.AdaptType(cmd[2]).(int)
	end, endOk := utils.AdaptType(cmd[3]).(int)

	if !startOk || !endOk {
		return nil, errors.New("start and end indices must be integers")
	}

	if end < start && end != -1 {
		return nil, errors.New("end index must be greater than start index or -1")
	}

	if !server.KeyExists(key) {
		return nil, errors.New("LTRIM command on non-list item")
	}

	if _, err = server.KeyLock(ctx, key); err != nil {
		return nil, err
	}
	defer server.KeyUnlock(key)

	list, ok := server.GetValue(key).([]interface{})
	if !ok {
		return nil, errors.New("LTRIM command on non-list item")
	}

	if !(start >= 0 && start < len(list)) {
		return nil, errors.New("start index must be within list boundary")
	}

	if end == -1 || end > len(list) {
		server.SetValue(ctx, key, list[start:])
		return []byte(utils.OK_RESPONSE), nil
	}

	server.SetValue(ctx, key, list[start:end])
	return []byte(utils.OK_RESPONSE), nil
}

func handleLRem(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := lremKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	key := keys[0]
	value := cmd[3]

	count, ok := utils.AdaptType(cmd[2]).(int)
	if !ok {
		return nil, errors.New("count must be an integer")
	}

	absoluteCount := utils.AbsInt(count)

	if !server.KeyExists(key) {
		return nil, errors.New("LREM command on non-list item")
	}

	if _, err = server.KeyLock(ctx, key); err != nil {
		return nil, err
	}
	defer server.KeyUnlock(key)

	list, ok := server.GetValue(key).([]interface{})
	if !ok {
		return nil, errors.New("LREM command on non-list item")
	}

	switch {
	default:
		// Count is zero, keep list the same
	case count > 0:
		// Start from the head
		for i := 0; i < len(list); i++ {
			if absoluteCount == 0 {
				break
			}
			if fmt.Sprintf("%v", list[i]) == value {
				list[i] = nil
				absoluteCount -= 1
			}
		}
	case count < 0:
		// Start from the tail
		for i := len(list) - 1; i >= 0; i-- {
			if absoluteCount == 0 {
				break
			}
			if fmt.Sprintf("%v", list[i]) == value {
				list[i] = nil
				absoluteCount -= 1
			}
		}
	}

	list = slices.DeleteFunc(list, func(elem interface{}) bool {
		return elem == nil
	})

	server.SetValue(ctx, key, list)

	return []byte(utils.OK_RESPONSE), nil
}

func handleLMove(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := lmoveKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	source := keys[0]
	destination := keys[1]
	whereFrom := strings.ToLower(cmd[3])
	whereTo := strings.ToLower(cmd[4])

	if !slices.Contains([]string{"left", "right"}, whereFrom) || !slices.Contains([]string{"left", "right"}, whereTo) {
		return nil, errors.New("wherefrom and whereto arguments must be either LEFT or RIGHT")
	}

	if !server.KeyExists(source) || !server.KeyExists(destination) {
		return nil, errors.New("both source and destination must be lists")
	}

	if _, err = server.KeyLock(ctx, source); err != nil {
		return nil, err
	}
	defer server.KeyUnlock(source)

	_, err = server.KeyLock(ctx, destination)
	if err != nil {
		return nil, err
	}
	defer server.KeyUnlock(destination)

	sourceList, sourceOk := server.GetValue(source).([]interface{})
	destinationList, destinationOk := server.GetValue(destination).([]interface{})

	if !sourceOk || !destinationOk {
		return nil, errors.New("both source and destination must be lists")
	}

	switch whereFrom {
	case "left":
		server.SetValue(ctx, source, append([]interface{}{}, sourceList[1:]...))
		if whereTo == "left" {
			server.SetValue(ctx, destination, append(sourceList[0:1], destinationList...))
		} else if whereTo == "right" {
			server.SetValue(ctx, destination, append(destinationList, sourceList[0]))
		}
	case "right":
		server.SetValue(ctx, source, append([]interface{}{}, sourceList[:len(sourceList)-1]...))
		if whereTo == "left" {
			server.SetValue(ctx, destination, append(sourceList[len(sourceList)-1:], destinationList...))
		} else if whereTo == "right" {
			server.SetValue(ctx, destination, append(destinationList, sourceList[len(sourceList)-1]))
		}
	}

	return []byte(utils.OK_RESPONSE), nil
}

func handleLPush(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := lpushKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	var newElems []interface{}

	for _, elem := range cmd[2:] {
		newElems = append(newElems, utils.AdaptType(elem))
	}

	key := keys[0]

	if !server.KeyExists(key) {
		switch strings.ToLower(cmd[0]) {
		case "lpushx":
			return nil, errors.New("LPUSHX command on non-list item")
		default:
			// TODO: Retry CreateKeyAndLock until we obtain the key lock
			if _, err = server.CreateKeyAndLock(ctx, key); err != nil {
				return nil, err
			}
			server.SetValue(ctx, key, []interface{}{})
		}
	} else {
		if _, err = server.KeyLock(ctx, key); err != nil {
			return nil, err
		}
	}
	defer server.KeyUnlock(key)

	currentList := server.GetValue(key)

	l, ok := currentList.([]interface{})
	if !ok {
		return nil, errors.New("LPUSH command on non-list item")
	}

	server.SetValue(ctx, key, append(newElems, l...))
	return []byte(utils.OK_RESPONSE), nil
}

func handleRPush(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := rpushKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	key := keys[0]

	var newElems []interface{}

	for _, elem := range cmd[2:] {
		newElems = append(newElems, utils.AdaptType(elem))
	}

	if !server.KeyExists(key) {
		switch strings.ToLower(cmd[0]) {
		case "rpushx":
			return nil, errors.New("RPUSHX command on non-list item")
		default:
			// TODO: Retry CreateKeyAndLock until we managed to obtain the key
			if _, err = server.CreateKeyAndLock(ctx, key); err != nil {
				return nil, err
			}
			server.SetValue(ctx, key, []interface{}{})
		}
	} else {
		if _, err = server.KeyLock(ctx, key); err != nil {
			return nil, err
		}
	}
	defer server.KeyUnlock(key)

	currentList := server.GetValue(key)

	l, ok := currentList.([]interface{})

	if !ok {
		return nil, errors.New("RPUSH command on non-list item")
	}

	server.SetValue(ctx, key, append(l, newElems...))
	return []byte(utils.OK_RESPONSE), nil
}

func handlePop(ctx context.Context, cmd []string, server utils.Server, conn *net.Conn) ([]byte, error) {
	keys, err := popKeyFunc(cmd)
	if err != nil {
		return nil, err
	}

	key := keys[0]

	if !server.KeyExists(key) {
		return nil, fmt.Errorf("%s command on non-list item", strings.ToUpper(cmd[0]))
	}

	if _, err = server.KeyLock(ctx, key); err != nil {
		return nil, err
	}
	defer server.KeyUnlock(key)

	list, ok := server.GetValue(key).([]interface{})
	if !ok {
		return nil, fmt.Errorf("%s command on non-list item", strings.ToUpper(cmd[0]))
	}

	switch strings.ToLower(cmd[0]) {
	default:
		server.SetValue(ctx, key, list[1:])
		return []byte(fmt.Sprintf("+%v\r\n\r\n", list[0])), nil
	case "rpop":
		server.SetValue(ctx, key, list[:len(list)-1])
		return []byte(fmt.Sprintf("+%v\r\n\r\n", list[len(list)-1])), nil
	}
}

func Commands() []utils.Command {
	return []utils.Command{
		{
			Command:           "lpush",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.FastCategory},
			Description:       "(LPUSH key value1 [value2]) Prepends one or more values to the beginning of a list, creates the list if it does not exist.",
			Sync:              true,
			KeyExtractionFunc: lpushKeyFunc,
			HandlerFunc:       handleLPush,
		},
		{
			Command:           "lpushx",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.FastCategory},
			Description:       "(LPUSHX key value) Prepends a value to the beginning of a list only if the list exists.",
			Sync:              true,
			KeyExtractionFunc: lpushKeyFunc,
			HandlerFunc:       handleLPush,
		},
		{
			Command:           "lpop",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.FastCategory},
			Description:       "(LPOP key) Removes and returns the first element of a list.",
			Sync:              true,
			KeyExtractionFunc: popKeyFunc,
			HandlerFunc:       handlePop,
		},
		{
			Command:           "llen",
			Categories:        []string{utils.ListCategory, utils.ReadCategory, utils.FastCategory},
			Description:       "(LLEN key) Return the length of a list.",
			Sync:              false,
			KeyExtractionFunc: llenKeyFunc,
			HandlerFunc:       handleLLen,
		},
		{
			Command:           "lrange",
			Categories:        []string{utils.ListCategory, utils.ReadCategory, utils.SlowCategory},
			Description:       "(LRANGE key start end) Return a range of elements between the given indices.",
			Sync:              false,
			KeyExtractionFunc: lrangeKeyFunc,
			HandlerFunc:       handleLRange,
		},
		{
			Command:           "lindex",
			Categories:        []string{utils.ListCategory, utils.ReadCategory, utils.SlowCategory},
			Description:       "(LINDEX key index) Gets list element by index.",
			Sync:              false,
			KeyExtractionFunc: lindexKeyFunc,
			HandlerFunc:       handleLIndex,
		},
		{
			Command:           "lset",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.SlowCategory},
			Description:       "(LSET key index value) Sets the value of an element in a list by its index.",
			Sync:              true,
			KeyExtractionFunc: lsetKeyFunc,
			HandlerFunc:       handleLSet,
		},
		{
			Command:           "ltrim",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.SlowCategory},
			Description:       "(LTRIM key start end) Trims a list to the specified range.",
			Sync:              true,
			KeyExtractionFunc: ltrimKeyFunc,
			HandlerFunc:       handleLTrim,
		},
		{
			Command:           "lrem",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.SlowCategory},
			Description:       "(LREM key count value) Remove elements from list.",
			Sync:              true,
			KeyExtractionFunc: lremKeyFunc,
			HandlerFunc:       handleLRem,
		},
		{
			Command:           "lmove",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.SlowCategory},
			Description:       "(LMOVE source destination <LEFT | RIGHT> <LEFT | RIGHT>) Move element from one list to the other specifying left/right for both lists.",
			Sync:              true,
			KeyExtractionFunc: lmoveKeyFunc,
			HandlerFunc:       handleLMove,
		},
		{
			Command:           "rpop",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.FastCategory},
			Description:       "(RPOP key) Removes and gets the last element in a list.",
			Sync:              true,
			KeyExtractionFunc: popKeyFunc,
			HandlerFunc:       handlePop,
		},
		{
			Command:           "rpush",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.FastCategory},
			Description:       "(RPUSH key value [value2]) Appends one or multiple elements to the end of a list.",
			Sync:              true,
			KeyExtractionFunc: rpushKeyFunc,
			HandlerFunc:       handleRPush,
		},
		{
			Command:           "rpushx",
			Categories:        []string{utils.ListCategory, utils.WriteCategory, utils.FastCategory},
			Description:       "(RPUSHX key value) Appends an element to the end of a list, only if the list exists.",
			Sync:              true,
			KeyExtractionFunc: rpushKeyFunc,
			HandlerFunc:       handleRPush,
		},
	}
}
