package main

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strings"
)

func main() {
	err := run()

	if err != nil {
		fmt.Printf("Error: %#+v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	reader := bufio.NewReader(os.Stdin)

	for true {
		line, err := reader.ReadString('\n')

		if err != nil {
			return err
		}

		err = runLine(line)

		if err != nil {
			fmt.Printf("Error: %#+v\n", err)
		}
	}

	return nil
}

func runLine(line string) error {
	reader := strings.NewReader(line)
	scanner := NewScanner(reader)
	parser := NewParser(scanner)

	cmd, err := parser.Parse()

	if err != nil {
		return err
	}

	result, err := executeCommand(cmd)

	if err != nil {
		return err
	}

	fmt.Printf("%#+v\n", result)

	return nil
}

func executeCommand(cmd *Command) (interface{}, error) {
	fn, ok := commands[cmd.Name]

	if !ok {
		return nil, fmt.Errorf("Unknown command %s", cmd.Name)
	}

	val := reflect.ValueOf(fn)
	typ := val.Type()

	if typ.NumIn() != 1 || typ.NumOut() != 2 {
		return nil, fmt.Errorf("Invalid function declaration")
	}

	if !typ.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, fmt.Errorf("Invalid function declaration")
	}

	argTyp := typ.In(0)

	if argTyp.Kind() == reflect.Ptr {
		argTyp = argTyp.Elem()
	}

	arg := reflect.New(argTyp)

	err := unmarshalCommand(arg, cmd)

	if err != nil {
		return nil, err
	}

	result := val.Call([]reflect.Value{arg})

	if !result[1].IsNil() {
		return nil, result[1].Interface().(error)
	}

	return result[0].Interface(), nil
}

func unmarshalCommand(val reflect.Value, cmd *Command) error {
	typ := val.Type()

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag, ok := field.Tag.Lookup("cmd")

		if !ok {
			tag = field.Name
		}

		values := strings.Split(tag, ",")
		name := values[0]

		if name == "" {
			err := unmarshalValue(val.Field(i), cmd.Primary)

			if err != nil {
				return err
			}
		} else {
			arg, ok := cmd.Parameters[name]

			if ok {
				err := unmarshalValue(val.Field(i), arg)

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func unmarshalValue(val reflect.Value, input interface{}) error {
	switch v := input.(type) {
	case *Command:
		result, err := executeCommand(v)

		if err != nil {
			return err
		}

		val.Set(reflect.ValueOf(result))
	case *Token:
		err := extractTokenValue(v, val.Addr().Interface())

		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Invalid token as parameter")
	}

	return nil
}

func extractTokenValue(t *Token, value interface{}) error {
	switch v := value.(type) {
	case *string:
		*v = t.Text
	default:
		return fmt.Errorf("Invalid struct type")
	}

	return nil
}
